package main

import (
	"carefront/libs/aws"
	"carefront/libs/aws/sqs"
	"encoding/json"
	"fmt"
	"time"
)

type erxMessage struct {
	PrescriptionId     int
	PrescriptionStatus string
}

func main() {
	awsAuth := aws.KeysFromEnvironment()

	awsClient := &aws.Client{
		Auth: awsAuth,
	}

	sq := &sqs.SQS{
		Region: aws.USEast,
		Client: awsClient,
	}

	queueUrl, err := sq.GetQueueUrl("erx", "")
	if err != nil {
		panic(err.Error())
	}

	go func() {
		for {
			prescriptionStatus := &erxMessage{}
			prescriptionStatus.PrescriptionId = 10
			prescriptionStatus.PrescriptionStatus = "TESTING"
			jsonData, err := json.Marshal(prescriptionStatus)
			if err != nil {
				panic(err.Error())
			}
			err = sq.SendMessage(queueUrl, 0, string(jsonData))
			if err != nil {
				panic(err.Error())
			}
			time.Sleep(10 * time.Second)
		}
	}()

	go func() {
		for {
			msgs, err := sq.ReceiveMessage(queueUrl, nil, 1, 5, 1)
			if err != nil {
				fmt.Println("Error receiving messages: " + err.Error())
				time.Sleep(10 * time.Second)
			}

			for _, msg := range msgs {
				prescriptionStatus := &erxMessage{}
				err = json.Unmarshal([]byte(msg.Body), prescriptionStatus)
				if err != nil {
					panic(err.Error())
				}
				fmt.Println("Receiving Message: " + prescriptionStatus.PrescriptionStatus)
				err = sq.DeleteMessage(queueUrl, msg.ReceiptHandle)
				if err != nil {
					fmt.Println("Error deleting message: " + err.Error())
					time.Sleep(10 * time.Second)
				}
				time.Sleep(10 * time.Second)
			}
		}
	}()

	time.Sleep(10 * time.Minute)
}
