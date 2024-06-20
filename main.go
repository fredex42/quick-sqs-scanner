package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/TylerBrock/colorjson"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func FindQueue(client *sqs.Client, name *string) (string, error) {
	var nextToken *string

	req := &sqs.ListQueuesInput{
		MaxResults:      aws.Int32(20),
		QueueNamePrefix: name,
		NextToken:       nextToken,
	}
	response, err := client.ListQueues(context.Background(), req)

	if err != nil {
		return "", err
	}

	if len(response.QueueUrls) == 0 {
		return "", errors.New("No queues found matching that name")
	}
	if len(response.QueueUrls) > 1 {
		log.Printf("Found %d queues matching '%s':", len(response.QueueUrls), *name)
		for _, url := range response.QueueUrls {
			log.Printf("\t%s", url)
		}
		if len(response.QueueUrls) == 20 { //we limited the request to 20
			log.Printf("There may be more than this, we hit the request limit.")
		}
		return "", errors.New("Please narrow down your search by providing a specific name")
	}

	return response.QueueUrls[0], nil
}

func Decode(msgBody *string) *string {
	if msgBody == nil {
		return nil
	}

	var content map[string]interface{}
	err := json.Unmarshal([]byte(*msgBody), &content)
	if err != nil {
		return msgBody
	}

	if snsMessage, haveSnsMessage := content["Message"]; haveSnsMessage {
		theMessage := snsMessage.(string)
		return &theMessage
	}
	return msgBody
}

func recursiveTruncate(obj *map[string]interface{}, currentField string, remainingFields []string, truncateAt *int) {
	if value, haveField := (*obj)[currentField]; haveField {
		if mapValue, valueIsMap := value.(map[string]interface{}); valueIsMap && len(remainingFields) > 0 {
			recursiveTruncate(&mapValue, remainingFields[0], remainingFields[1:], truncateAt)
		}
		if stringValue, valueIsString := value.(string); valueIsString && len(stringValue) > *truncateAt {
			(*obj)[currentField] = fmt.Sprintf("%s… (%d chars)", stringValue[0:*truncateAt], len(stringValue))
		}
	}
}

func PrettyPrint(f *colorjson.Formatter, msg *string, truncateFields *[]string, truncateAt *int) {
	var obj map[string]interface{}
	err := json.Unmarshal([]byte(*msg), &obj)

	if truncateFields != nil {
		for _, field := range *truncateFields {
			fieldParts := strings.Split(field, ".")
			var tail []string
			if len(fieldParts) > 1 {
				tail = fieldParts[1:]
			}
			recursiveTruncate(&obj, fieldParts[0], tail, truncateAt)
		}
	}

	if err == nil {
		str, _ := colorjson.Marshal(obj)
		fmt.Println(string(str))
	} else {
		fmt.Println(*msg)
	}
}

func main() {
	queueName := flag.String("queue", "", "name of the queue you want to listen to")
	truncateFieldsStr := flag.String("truncate", "", "optionally set a list of field names to be truncated in the output. Separate with commas and don't pad with whitespace. If you need to specify multiple levels, separate them with a . (e.g. `details.event`)")
	truncateAt := flag.Int("truncateAt", 36, "truncate string fields over this length, if they are listed in `truncate`")
	flag.Parse()

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("Unable to load AWS SDK config: %v", err)
	}

	client := sqs.NewFromConfig(cfg)

	url, err := FindQueue(client, queueName)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Polling %s, press CTRL-C to exit", url)

	jsonFormatter := colorjson.NewFormatter()

	var truncateFieldsPtr *[]string
	if *truncateFieldsStr != "" {
		temp := strings.Split(*truncateFieldsStr, ",")
		truncateFieldsPtr = &temp
	}

	for {
		req := &sqs.ReceiveMessageInput{
			QueueUrl:        &url,
			WaitTimeSeconds: 10,
		}
		messages, err := client.ReceiveMessage(context.Background(), req)
		if err != nil {
			log.Print("Unable to poll: ", err)
			os.Exit(1)
		}

		for _, msg := range messages.Messages {
			if msg.Body != nil {
				decoded := Decode(msg.Body)
				PrettyPrint(jsonFormatter, decoded, truncateFieldsPtr, truncateAt)
			}
			client.DeleteMessage(context.Background(), &sqs.DeleteMessageInput{
				QueueUrl:      &url,
				ReceiptHandle: msg.ReceiptHandle,
			})
		}
	}
}
