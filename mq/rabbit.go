package mq

import (
	"github.com/streadway/amqp"
	"go.uber.org/zap"
	"grpc_tls/log"
)

type RabbitMQ struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	QueueName    string //队列名称
	ExchangeName string //交换机
	RouteKey     string //路由key
	MqUrl        string //链接信息
}

var mqUrl = "amqp://weq:930403@127.0.0.1:5672/"

// NewRabbitMQ 创建队列服务
func NewRabbitMQ(queue, exchange, router string) *RabbitMQ {
	var err error
	var rabbitmq = &RabbitMQ{
		QueueName:    queue,
		ExchangeName: exchange,
		RouteKey:     router,
		MqUrl:        mqUrl,
	}

	rabbitmq.conn, err = amqp.Dial(rabbitmq.MqUrl)
	if err != nil {
		log.Logger.Fatal("amqp.Dial err", zap.Error(err))
		return rabbitmq
	}

	rabbitmq.channel, err = rabbitmq.conn.Channel() //设置信道
	if err != nil {
		log.Logger.Fatal("conn.Channel err", zap.Error(err))
		return rabbitmq
	}

	err = rabbitmq.channel.ExchangeDeclare(
		"ratmp",            //交换机名称
		amqp.ExchangeTopic, //交换机类型: [direct,fanout,topic,headers]
		true,               //是否持久化: false重启服务器消息消失, 通常设置为false
		true,               //是否自动删除
		false,              // true表示不能将同一个conn中发送的消息传递给同个conn中的消费者
		false,              //是否阻塞: true不阻塞， false阻塞, true等消费后在进来下一个消息
		nil,                //额外参数
	)
	if err != nil {
		log.Logger.Fatal("channel.ExchangeDeclare err", zap.Error(err))
		return rabbitmq
	}

	return rabbitmq
}

// Close rabbitmq关闭服务
func (r *RabbitMQ) Close() {
	if err := r.channel.Close(); err != nil {
		log.Logger.Fatal("rabbitmq channel.Close err", zap.Error(err))
		return
	}

	if err := r.conn.Close(); err != nil {
		log.Logger.Fatal("rabbitmq conn.Close err", zap.Error(err))
	}
}

// NewRabbitMQSimple 简单模式step：1.创建简单模式下rabbitmq实列
func NewRabbitMQSimple(queue string) *RabbitMQ {
	//simple模式是rabbitmq最简单的一种模式， 未设置交换机，会使用默认的default交换机
	return NewRabbitMQ(queue, "", "")
}

// PublishSimple 简单模式step：2.简单模式下生产代码
func (r *RabbitMQ) PublishSimple(message string) {
	//1.申请队列，不存在创建， 存在直接使用
	_, err := r.channel.QueueDeclare(
		r.QueueName, true, false,
		false, false, nil)

	if err != nil {
		log.Logger.Error("PublishSimple", zap.Error(err))
	}

	//2.发送消息到队列当中
	err = r.channel.Publish(
		//交换机，simple模式下为空，空模式使用default交换机
		r.ExchangeName,
		r.QueueName,
		//true会根据exchange和routerKey规则， 无法找到符合条件的队列会把发送的消息返回给发送者
		false,
		//为true时，当exchange发送消息队列后， 队列没有绑定消费者会把消息返回给发送者
		false,
		//消息体
		amqp.Publishing{
			//消息内容持久化，本项配置很关键
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         []byte(message),
		})

	if err != nil {
		log.Logger.Error("PublishSimple => channel.Publish", zap.Error(err))
	}
}

// ConsumeSimple 简单模式step：3.简单模式下消费者代码
func (r *RabbitMQ) ConsumeSimple() {
	//1.申请队列， 不存在创建，存在直接使用
	_, err := r.channel.QueueDeclare(
		r.QueueName,
		//是否持久化，未消费就在队列里边，重启服务器数据会丢失， 通常设置为false
		true,
		//是否自动删除， 最后一个消费者断开后是否将消息删除， 默认设置为false
		false,
		//是否具有排他性
		false,
		//是否阻塞，发送消息后需要等到消费后下一个消息才能进来， 跟go无缓冲channel一样， 默认设置为false
		false,
		nil)
	if err != nil {
		log.Logger.Error("channel.QueueDeclare", zap.Error(err))
	}

	//限流：流量控制，防止数据库暴库
	_ = r.channel.Qos(
		//队列每次只消费一个消息，处理完成后才会在发送第二个
		1,
		//服务传递得最大容量
		0,
		//true对channel可用，false只对当前队列可用
		false)

	//2.接收消息, 建立socket链接，会一直监听，不会终止
	msgs, err := r.channel.Consume(
		//队列名称
		r.QueueName,
		//用来区别多个消费者
		"",
		//是否自动应答：消息被消费主动通知服务端， 服务端删除被消费的消息， 默认设置为true
		true,
		//是否具有排他性
		false,
		//true表示不能将同一个connection中发送的消息传递给同个connection中的消费者
		false,
		//消费是否阻塞：true不阻塞， false阻塞
		false,
		nil)
	if err != nil {
		log.Logger.Error("channel.Consume", zap.Error(err))
	}

	//3.消费消息
	go func() {
		for d := range msgs {
			log.Logger.Warn("recv a message", zap.Binary("data", d.Body))
			//表示确认当前消息， true表示确认所有未确认的消息
			if err = d.Ack(false); err != nil {
				log.Logger.Error("d.Ack 消息确认失败",
					zap.Binary("data", d.Body),
					zap.Error(err),
				)
			}
		}
	}()

	log.Logger.Info("[*] Waiting for message, To exit press CTRL+C")

	//一直阻塞，防止主程序死掉
	var forever = make(chan bool)
	defer close(forever)
	<-forever
}

// DeadLetterMQ 创建死信队列交换机
func DeadLetterMQ(exchange string) *RabbitMQ {
	return NewRabbitMQ("", exchange, "")
}

// DeadLetterQueue 死信队列生产者
func (r *RabbitMQ) DeadLetterQueue(message string) {
	//创建交换机
	err := r.channel.ExchangeDeclare(
		r.ExchangeName,      //交换机名称
		amqp.ExchangeDirect, //类型:[direct,fanout,topic,headers]
		true,
		false, //消息不会自动删除
		false,
		false,
		nil)

	if err != nil {
		log.Logger.Error("DeadLetterQueue", zap.Error(err))
		return
	}

	//发送消息
	err = r.channel.Publish(
		r.ExchangeName,
		r.QueueName,
		false,
		false,
		amqp.Publishing{
			Headers:      amqp.Table{},
			ContentType:  "text/plain",
			Body:         []byte(message),
			DeliveryMode: amqp.Persistent, //持久化
			Priority:     0,
		},
	)

	if err != nil {
		log.Logger.Error("消息发生失败",
			zap.Error(err),
			zap.String("data", message))
	}
}

// DeadLetterConsumer 死信队列正常消费消息
func (r *RabbitMQ) DeadLetterConsumer() {
	var err = r.channel.ExchangeDeclare(
		r.ExchangeName,
		amqp.ExchangeDirect,
		true,
		false,
		false,
		false,
		nil)
	if err != nil {
		log.Logger.Fatal("声明交换机失败", zap.Error(err))
		return
	}

	//添加死信队列交换器属性
	var args = map[string]interface{}{
		//交换机名称
		"x-dead-letter-exchange": r.ExchangeName,
		//路由key,不指定使用队列路由键
		"x-dead-letter-routing-key": r.RouteKey,
		//过期时间,单位毫秒
		"x-message-ttl": 6000,
	}
	//声明队列
	_, err = r.channel.QueueDeclare(
		r.QueueName,
		true,
		false,
		false,
		false,
		args)
	if err != nil {
		log.Logger.Fatal("声明队列失败", zap.Error(err))
		return
	}

	//绑定交换器/队列和key
	err = r.channel.QueueBind(
		r.QueueName,
		r.RouteKey,
		//交换机名称
		r.ExchangeName,
		false,
		nil)
	if err != nil {
		log.Logger.Fatal("绑定交换机失败", zap.Error(err))
		return
	}

	//开启推模式消费
	var delvers, err01 = r.channel.Consume(
		r.QueueName,
		"",
		false,
		false,
		false,
		false,
		nil)
	if err01 != nil {
		log.Logger.Fatal("Consume", zap.Error(err))
		return
	}

	for true {
		select {
		case message, ok := <-delvers:
			if ok {
				//确认收到的消息
				if err = message.Ack(true); err != nil {
					//过期时间内未进行确认,此消息会流入死信队列, 此时进行消息确认就会报错
					log.Logger.Error("message.Ack", zap.Error(err))
					return
				}
			}
		}
	}
}

// ConsumerDlx 消费死信队列
func (r *RabbitMQ) ConsumerDlx() {
	//申明交换机
	var err = r.channel.ExchangeDeclare(
		r.ExchangeName,
		amqp.ExchangeFanout,
		true,  //持久化
		false, //自动删除
		//是否内置交换器,只能通过交换器将消息路由到此交互器，不能通过客户端发送消息
		false, false, nil)
	if err != nil {
		log.Logger.Fatal("ConsumerDlx", zap.Error(err))
		return
	}

	//申明队列
	_, err = r.channel.QueueDeclare(
		r.QueueName, true, false,
		false,
		//是否等待服务器返回OK
		false, nil)
	if err != nil {
		log.Logger.Fatal("QueueDeclare", zap.Error(err))
		return
	}

	//将交换器和队列/路由key绑定
	err = r.channel.QueueBind(
		r.QueueName, "",
		r.ExchangeName, false, nil)
	if err != nil {
		log.Logger.Fatal("QueueBind", zap.Error(err))
		return
	}

	//开启推送模式
	var delvers, err01 = r.channel.Consume(
		r.QueueName, "", false,
		false, false, false, nil)
	if err01 != nil {
		log.Logger.Fatal("Consume", zap.Error(err))
		return
	}

	for true {
		select {
		case message, ok := <-delvers:
			if ok {
				//确认接受到的消息
				if err = message.Ack(true); err != nil {
					log.Logger.Error("message.Ack", zap.Error(err))
				}
			}
		}
	}
}

// NewRabbitMQPubSub 订阅模式step1：创建rabbitmq实例
func NewRabbitMQPubSub(exchangeName string) *RabbitMQ {
	//创建RabbitMQ实列
	return NewRabbitMQ("", exchangeName, "")
}

// PublishPub 订阅模式step：2生产者
func (r *RabbitMQ) PublishPub(message string) {
	//1.创建交换机，已存在不创建，不存在即创建
	err := r.channel.ExchangeDeclare(
		r.ExchangeName,
		//广播类型， 订阅者模式下我们需要将类型设置为广播类型
		amqp.ExchangeFanout,
		//消息是否持久化， false服务器重启，消息就没了，通常默认设置false
		true,
		//是否为自动删除
		//1.交换机自动删除的条件，有队列或者交换器绑定了本交换器，然后所有队列或交换器都与本交换器解除绑定，true时，此交换器就会被自动删除
		//2.队列自动删除的条件，有消息者订阅本队列，然后所有消费者都解除订阅此队列，true时，此队列会自动删除，即使此队列中还有消息
		false,
		//true表示这个交换机不可以被客户端推送消息，仅用来交换机和交换机直接的绑定
		false,
		false,
		nil)
	if err != nil {
		log.Logger.Fatal("ExchangeDeclare", zap.Error(err))
		return
	}

	//2.发送消息
	err = r.channel.Publish(
		r.ExchangeName,
		"",
		//true时会根据exchange和routeKey规则，如果没有符合条件的队列会把消息返还给发送者
		false,
		//true时当exchange发送消息到队列如果改队列没有绑定消费者会把消息返还给发送者
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(message),
		})
	if err != nil {
		log.Logger.Fatal("消失发送失败", zap.String("data", message))
	}
}

func (r *RabbitMQ) RecieveSub() {
	//创建交换机，存在直接使用
	err := r.channel.ExchangeDeclare(
		r.ExchangeName,
		//广播类型 订阅模式下需要将其设置为广播类型
		amqp.ExchangeFanout,
		//消息是否持久化， false服务器重启，消息就没了，通常默认设置false
		true,
		//是否为自动删除
		//1.交换机自动删除的条件，有队列或者交换器绑定了本交换器，然后所有队列或交换器都与本交换器解除绑定，true时，此交换器就会被自动删除
		//2.队列自动删除的条件，有消息者订阅本队列，然后所有消费者都解除订阅此队列，true时，此队列会自动删除，即使此队列中还有消息
		false,
		//true表示这个交换机不可以被客户端推送消息，仅用来交换机和交换机直接的绑定
		false,
		//队列是否阻塞：true不阻塞， false阻塞
		false,
		nil)
	if err != nil {
		log.Logger.Fatal("RecieveSub", zap.Error(err))
		return
	}

	//2.试探性创建队列，这里注意不要写队列名称
	q, err := r.channel.QueueDeclare(
		//随机队列名称， 这个地方一定要留空
		"",
		false,
		false,
		//具体有排他性：即队列只为当前链接服务 链接断开 队列被删除
		true,
		false,
		nil)

	//3.绑定队列到交换机中去
	err = r.channel.QueueBind(
		q.Name,
		//在pub/sub模式下，这里的key要为空
		"",
		r.ExchangeName,
		false,
		nil)

	//4.消费代码
	message, err := r.channel.Consume(
		//队列名称
		r.QueueName,
		//用来区分多个消费者
		"",
		//是否自动应答， 当消息被消费后主动通知服务器，服务将消息删除， 默认设置为true
		true,
		//是否具有排他性
		false,
		//如果为true表示不能将同一个connection中发送的消息传递给同个connection中的消费者
		false,
		//是否阻塞： true不阻塞， false阻塞
		false,
		nil)
	if err != nil {
		log.Logger.Error("Consume", zap.Error(err))
		return
	}

	var forever = make(chan bool)
	defer close(forever)
	go func() {
		for d := range message {
			log.Logger.Debug("recv a message", zap.Binary("data", d.Body))
		}
	}()

	log.Logger.Warn("退出请按 CTRL+C")
	<-forever
}

// NewRabbitMQRouting 路由模式step:1创建RabbitMQ实列
func NewRabbitMQRouting(exchange string, router string) *RabbitMQ {
	return NewRabbitMQ("", exchange, router)
}

// PublishRouting 路由模式step:2发送消息
func (r *RabbitMQ) PublishRouting(message string) {
	//1.尝试创建交换机
	err := r.channel.ExchangeDeclare(
		r.ExchangeName,
		//类型：路由模式需要将其设置为 direct， 和订阅模式不一样
		amqp.ExchangeDirect,
		//是否持久化， false重启服务器消息就没了，通常设置为false
		true,
		//是否为自动删除
		//1.交换机自动删除的条件，有队列或者交换器绑定了本交换器，然后所有队列或交换器都与本交换器解除绑定，true时，此交换器就会被自动删除
		//2.队列自动删除的条件，有消息者订阅本队列，然后所有消费者都解除订阅此队列，true时，此队列会自动删除，即使此队列中还有消息
		false,
		//如果为true表示不能将同一个connection中发送的消息传递给同个connection中的消费者
		false,
		//是否阻塞： true不阻塞， false阻塞
		false,
		nil)

	if err != nil {
		log.Logger.Fatal("PublishRouting", zap.Error(err))
		return
	}

	//2.发送消息
	err = r.channel.Publish(
		//设置交换机
		r.ExchangeName,
		//设置交换机绑定的routeing
		r.RouteKey,
		//true时， 当交换机发送消息到队列上发现队列没有绑定消费者，会把消息返回给发送者
		false,
		false,
		//发送消息
		amqp.Publishing{
			ContentType: "text/Plain",
			Body:        []byte(message),
		})

	if err != nil {
		log.Logger.Error("Publish", zap.Error(err))
	}
}

// RecieveRouting 路由模式step：3消费者
func (r *RabbitMQ) RecieveRouting() {
	//创建交换机，不存在创建，存在直接使用
	err := r.channel.ExchangeDeclare(
		r.ExchangeName,
		//类型：路由模式下必须设置为direct
		amqp.ExchangeDirect,
		//是否持久化， false重启服务器消息就没了，通常设置为false
		true,
		//是否为自动删除
		//1.交换机自动删除的条件，有队列或者交换器绑定了本交换器，然后所有队列或交换器都与本交换器解除绑定，true时，此交换器就会被自动删除
		//2.队列自动删除的条件，有消息者订阅本队列，然后所有消费者都解除订阅此队列，true时，此队列会自动删除，即使此队列中还有消息
		false,
		//如果为true表示不能将同一个connection中发送的消息传递给同个connection中的消费者
		false,
		//是否阻塞： true不阻塞， false阻塞
		false,
		nil)

	if err != nil {
		log.Logger.Error("RecieveRouting", zap.Error(err))
		return
	}

	//试探性创建队列，这里注意队列名称不要填写
	q, err := r.channel.QueueDeclare(
		//随机生产队列名称， 一定要留空
		"",
		false,
		false,
		//具体有排他性：即队列只为当前链接服务 链接断开 队列被删除
		true,
		false,
		nil)
	if err != nil {
		log.Logger.Fatal("QueueDeclare", zap.Error(err))
		return
	}

	//3.绑定队列到exchange中去
	err = r.channel.QueueBind(
		//队列名称， 通过key去找绑定好的队列
		q.Name,
		//在路由模式下，这里的key一定要填写
		r.RouteKey,
		r.ExchangeName,
		false,
		nil)

	//4.消费代码
	//4.1接受队列消息
	message, err := r.channel.Consume(
		q.Name,
		//用来区分消费者
		"",
		//是否自动应答， 当消息被消费后主动通知服务器，服务将消息删除， 默认设置为true
		true,
		//是否具有排他性
		false,
		//如果为true表示不能将同一个conn中发送的消息传递给同个conn中的消费者
		false,
		//是否阻塞： true不阻塞， false阻塞
		false,
		nil)

	if err != nil {
		log.Logger.Fatal("Consume", zap.Error(err))
	}

	var forever = make(chan bool)
	defer close(forever)
	go func() {
		for d := range message {
			log.Logger.Debug("recv", zap.Binary("data", d.Body))
		}
	}()

	log.Logger.Warn("退出请按 CTRL+C")
	<-forever
}

// NewRabbitMQTopic topic主题模式step：1创建Rabbit实例
func NewRabbitMQTopic(exchangeName, routerKey string) *RabbitMQ {
	return NewRabbitMQ("", exchangeName, routerKey)
}

// PublishTopic topic主题模式2:发送消息
func (r *RabbitMQ) PublishTopic(message string) {
	//1.创建交换机， 不存在创建，存在直接使用
	err := r.channel.ExchangeDeclare(
		r.ExchangeName,
		//类型： topic主题模式下，需将类型设置为topic
		amqp.ExchangeTopic,
		//是否持久化， false重启服务器消息就没了，通常设置为false
		true,
		//是否具有排他性
		false,
		//如果为true表示不能将同一个connection中发送的消息传递给同个connection中的消费者
		false,
		//是否阻塞： true不阻塞， false阻塞
		false,
		nil)
	if err != nil {
		log.Logger.Fatal("PublishTopic", zap.Error(err))
		return
	}

	//2.发送消息
	err = r.channel.Publish(
		//设置交换机
		r.ExchangeName,
		//交换机绑定到routeing
		r.RouteKey,
		//true时， 当交换机发送消息到队列上发现队列没有绑定消费者，会把消息返回给发送者
		false,
		false,
		//发送消息
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(message),
		})

	if err != nil {
		log.Logger.Error("Publish", zap.Error(err))
	}
}

// RecieveTopic topic主题模式step：消费者
// 需要注意key规则
// * 用于匹配一个单词， # 用于匹配多个单词（可以是零个）
// 匹配 huxiaobai.* 表示匹配huxiaobai.hello 但是huxaiobai.one.tow需要使用huxiaobai.#才能匹配
func (r *RabbitMQ) RecieveTopic() {
	//创建交换机，不存在创建，存在直接使用
	err := r.channel.ExchangeDeclare(
		//交换机名称
		r.ExchangeName,
		//类型， topic模式下，类型必须设置为topic
		amqp.ExchangeTopic,
		//是否持久化， false重启服务器消息就没了，通常设置为false
		true,
		//是否为自动删除
		//1.交换机自动删除的条件，有队列或者交换器绑定了本交换器，然后所有队列或交换器都与本交换器解除绑定，true时，此交换器就会被自动删除
		//2.队列自动删除的条件，有消息者订阅本队列，然后所有消费者都解除订阅此队列，true时，此队列会自动删除，即使此队列中还有消息
		false,
		//如果为true表示不能将同一个conn中发送的消息传递给同个conn中的消费者
		false,
		//是否阻塞： true不阻塞， false阻塞
		false,
		nil)
	if err != nil {
		log.Logger.Fatal("RecieveTopic", zap.Error(err))
		return
	}

	//2.试探性创建队列， 注意这里不要写队列名称
	q, err := r.channel.QueueDeclare(
		//不要填写队列名称， 随机生成队列名称
		"",
		false,
		//具体有排他性：即队列只为当前链接服务 链接断开 队列被删除
		false,
		true,
		false,
		nil)
	if err != nil {
		log.Logger.Fatal("QueueDeclare", zap.Error(err))
		return
	}

	err = r.channel.QueueBind(
		//队列名称
		q.Name,
		//路由模式下，topic模式下，这里routing要填写
		r.RouteKey,
		//交换机
		r.ExchangeName,
		false,
		nil)
	if err != nil {
		log.Logger.Fatal("QueueBind", zap.Error(err))
		return
	}

	//消费者代码
	//4.1接受队列消息
	message, err := r.channel.Consume(
		//队列名称
		q.Name,
		//用来区分消费者
		"",
		//是否具有排他性
		true,
		//是否自动应答， 当消息被消费后主动通知服务器，服务将消息删除， 默认设置为true
		false,
		//如果为true表示不能将同一个connection中发送的消息传递给同个connection中的消费者
		false,
		//是否阻塞： true不阻塞， false阻塞
		false,
		nil)
	if err != nil {
		log.Logger.Fatal("Consume", zap.Error(err))
		return
	}

	//4.2 真正开始消费消息
	var forever = make(chan bool)
	defer close(forever)
	go func() {
		for d := range message {
			log.Logger.Debug("recv a message", zap.Binary("data", d.Body))

			//当autoAck设置为true时， 需要手动告诉服务端，消费已被消费
			if err = d.Ack(false); err != nil {
				log.Logger.Error("d.Ack", zap.Error(err))
			}
		}
	}()

	log.Logger.Warn("退出请按 CTRL+C ")

	<-forever
}
