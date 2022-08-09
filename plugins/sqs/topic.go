//go:build sqs
// +build sqs

package sqs

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	aws_session "github.com/aws/aws-sdk-go/aws/session"
	aws_sqs "github.com/aws/aws-sdk-go/service/sqs"
	aws_sqsiface "github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/shoplineapp/go-app/plugins"
	"github.com/shoplineapp/go-app/plugins/env"
)

type Topic struct {
	Name string
	Arn  string

	AwsReceiveMessageInput *aws_sqs.ReceiveMessageInput

	aws_sqsiface.SQSAPI
}

type AwsTopicManager struct {
	Region    string
	TopicMaps map[string]*Topic
}

func init() {
	plugins.Registry = append(plugins.Registry, NewAwsTopicManager)
}

func NewAwsTopicManager(env *env.Env) *AwsTopicManager {
	topicMaps := make(map[string]*Topic)
	mgr := &AwsTopicManager{
		TopicMaps: topicMaps,
		Region:    env.GetEnv("AWS_REGION"),
	}

	return mgr
}

func (t *AwsTopicManager) SetRegion(region string) {
	t.Region = region
}

func (t *AwsTopicManager) AddTopic(name, arn string) *Topic {
	awsConfig := aws.Config{Region: aws.String(t.Region)}
	session := aws_session.Must(aws_session.NewSession(&awsConfig))

	topic := &Topic{
		Name:   name,
		Arn:    arn,
		SQSAPI: aws_sqs.New(session),
	}

	t.TopicMaps[name] = topic

	return topic
}

func (t *AwsTopicManager) RemoveTopic(name string) {
	delete(t.TopicMaps, name)
}

func (t *AwsTopicManager) GetTopic(name string) *Topic {
	return t.TopicMaps[name]
}

func (t *AwsTopicManager) GetTopics() []*Topic {
	m := make([]*Topic, 0, len(t.TopicMaps))
	for _, val := range t.TopicMaps {
		m = append(m, val)
	}

	return m
}

func ConvertStructToJson(object interface{}) string {
	var jsonData []byte
	jsonData, jsonErr := json.Marshal(object)
	if jsonErr != nil {
		panic(jsonErr)
	}

	return string(jsonData)
}
