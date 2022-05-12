package test_test

import (
	"grpc_tls/log"
	"grpc_tls/mq"
	"strconv"
	"testing"
	"time"
)

func init() {
	log.IniLog("./logs", -1)
}

//简单模式生产者
func BenchmarkSimpleProduce(b *testing.B) {
	var simple01 = mq.NewRabbitMQSimple("queue_one")
	var simple02 = mq.NewRabbitMQSimple("queue_two")

	simple02.PublishSimple("hello, world")
	for i := 0; i < 100; i++ {
		simple01.PublishSimple(time.Now().String())
		<-time.After(time.Second)
	}
}

//简单模式消费者
func BenchmarkSimpleConsumer(b *testing.B) {
	var consumer01 = mq.NewRabbitMQSimple("queue_one")
	go consumer01.ConsumeSimple()

	var consumer02 = mq.NewRabbitMQSimple("queue_one")
	go consumer02.ConsumeSimple()

	var consumer03 = mq.NewRabbitMQSimple("queue_two")
	consumer03.ConsumeSimple()
}

//订阅模式生产者
func BenchmarkPubSub(b *testing.B) {

	rabbitmq := mq.NewRabbitMQPubSub("newProduct")
	for i := 0; i < 100; i++ {
		var msg = "订阅模式生产第 " + strconv.Itoa(i) + " 条数据"
		rabbitmq.PublishPub(msg)

		<-time.After(time.Second * 3)
	}
}

//订阅模式消费者
func BenchmarkRecieveSubConsumer(b *testing.B) {
	var consumer01 = mq.NewRabbitMQPubSub("newProduct")
	go consumer01.RecieveSub()

	var consumer02 = mq.NewRabbitMQPubSub("newProduct")
	consumer02.RecieveSub()
}

//路由模式: 生产者
func BenchmarkRoutingProduce(b *testing.B) {
	//路由模式下通过key 将队列绑定到交换机上， 这个队列是内部自动生产的，不需要指定名称，使用时传递key即可找到绑定的队列

	var mq01 = mq.NewRabbitMQRouting("exHxb", "test_one_01")
	var mq02 = mq.NewRabbitMQRouting("exHxb", "test_one_02")

	for i := 0; i < 10; i++ {
		mq01.PublishRouting("hello 01 - " + time.Now().String())
		mq02.PublishRouting("hello 02 - " + time.Now().String())

		<-time.After(time.Second * 3)
	}
}

//路由模式消费者
func BenchmarkRoutingConsumer(b *testing.B) {
	var consumer01 = mq.NewRabbitMQRouting("exHxb", "test_one_01") //消费者01
	var consumer02 = mq.NewRabbitMQRouting("exHxb", "test_one_02") //消费者02

	go consumer01.RecieveRouting()
	consumer02.RecieveRouting()
}

//主题模式生产者
func BenchmarkTopicProduct(b *testing.B) {
	var topic01 = mq.NewRabbitMQTopic("hxbExc", "test_one")
	var topic02 = mq.NewRabbitMQTopic("hxbExc", "test_one.bak")

	for i := 0; i < 10; i++ {
		topic01.PublishTopic("hello one " + time.Now().String())
		topic02.PublishTopic("hello two " + time.Now().String())

		<-time.After(time.Second * 3)
	}
}

//主题模式消费者
func BenchmarkTopicConsumer(b *testing.B) {
	//#表示匹配多个单词， 也就是hxbExc交换机里边的所有消息
	var topicAll = mq.NewRabbitMQTopic("hxbExc", "#")
	go topicAll.RecieveTopic()

	//这里只匹配 test_one. 后边只能一个单词的key,通过这个key找到相应的队列
	var topic = mq.NewRabbitMQTopic("hxbExc", "test_one.*.bak")
	topic.RecieveTopic()
}
