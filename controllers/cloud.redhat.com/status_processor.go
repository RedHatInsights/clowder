package controllers

import (
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	apps "k8s.io/api/apps/v1"
)

//StatusCondition represents a condition and is able to convert
//conditions encoded in different ways for different status providers
type StatusCondition struct {
	Type   string
	Status string
}

//Public method to extract conditions from apps.DeploymentCondition
//and then convert and set those conditions
func (sc StatusCondition) FromDeployment(d apps.DeploymentCondition) {
	sc.setDefaults()
	sc.Type = string(d.Type)
	sc.Status = string(d.Status)
}

//Public method to set conditions based on string pointers
//from Kafka. We nil check these before using them.
func (sc StatusCondition) FromKafka(Type *string, Status *string) {
	sc.setDefaults()
	if Type != nil {
		sc.Type = *Type
	}
	if Status != nil {
		sc.Status = *Status
	}
}

//Private method to set default target values
func (sc StatusCondition) setDefaults() {
	none := "NONE"
	sc.Type = none
	sc.Status = none
}

//StatusHandler abstracts away various sources
//to provide a single API and logic for status evaluation
type StatusProcessor struct {
	Generation         int64
	ObservedGeneration int64
	Conditions         []StatusCondition
	TypeTarget         string
	StatusTarget       string
}

//Public method to convert k8s app.Deployment data for use in StatusProcessor
func (s StatusProcessor) ProcessDeployment(deployment apps.Deployment) {
	s.setTargets("Available", "True")
	s.Generation = deployment.Generation
	s.ObservedGeneration = deployment.Status.ObservedGeneration
	for _, deploymentCondition := range deployment.Status.Conditions {
		ca := StatusCondition{}
		ca.FromDeployment(deploymentCondition)
		s.Conditions = append(s.Conditions, ca)
	}
}

//Public method to convert strimzi.Kafka data for use in StatusProcessor
func (s StatusProcessor) ProcessKafka(kafka strimzi.Kafka) {
	s.setTargets("Ready", "True")
	s.setKafkaGenerations(kafka.Generation, kafka.Status.ObservedGeneration)
	for _, kafkaCondition := range kafka.Status.Conditions {
		s.convertAndAppendKafkaCondition(kafkaCondition.Type, kafkaCondition.Status)
	}
}

//Public method to convert strimzi.KafkaConnect data for use in StatusProcessor
func (s StatusProcessor) ProcessKafkaConnect(kafka strimzi.KafkaConnect) {
	s.setTargets("Ready", "True")
	s.setKafkaGenerations(kafka.Generation, kafka.Status.ObservedGeneration)
	for _, kafkaCondition := range kafka.Status.Conditions {
		s.convertAndAppendKafkaCondition(kafkaCondition.Type, kafkaCondition.Status)
	}
}

//Public method to convert strimzi.KafkaTopic data for use in StatusProcessor
func (s StatusProcessor) ProcessKafkaTopic(kafka strimzi.KafkaTopic) {
	s.setTargets("Ready", "True")
	s.setKafkaGenerations(kafka.Generation, kafka.Status.ObservedGeneration)
	for _, kafkaCondition := range kafka.Status.Conditions {
		s.convertAndAppendKafkaCondition(kafkaCondition.Type, kafkaCondition.Status)
	}
}

//Public method to get a boolean status
func (s StatusProcessor) GetStatus() bool {
	GenerationEqual := s.Generation <= s.ObservedGeneration
	ConditionsGood := true
	for _, condition := range s.Conditions {
		ConditionsGood = condition.Type == s.TypeTarget && condition.Status == s.StatusTarget
		if !ConditionsGood {
			break
		}
	}
	return GenerationEqual && ConditionsGood
}

//Private method to convert a kafka condition to a StatusCondition and then
//add that StatusCondition to the Conditions slice
func (s StatusProcessor) convertAndAppendKafkaCondition(Type *string, Status *string) {
	ca := StatusCondition{}
	ca.FromKafka(Type, Status)
	s.Conditions = append(s.Conditions, ca)
}

//Private method to set the generation and observed generation
//based on kafka source data
func (s StatusProcessor) setKafkaGenerations(generation int64, observedGeneration *int32) {
	s.Generation = generation
	if observedGeneration != nil {
		s.ObservedGeneration = int64(*observedGeneration)
	}
}

//Private method to set the type and status targets
func (s StatusProcessor) setTargets(typeTarget string, statusTarget string) {
	s.TypeTarget = typeTarget
	s.StatusTarget = statusTarget
}
