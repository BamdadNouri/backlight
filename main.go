package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/fvbock/endless"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type WebHookReq struct {
	Key    string `json:"key"`
	Color  string `json:"color"`
	Action string `json:"action"`
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connect lost: %v", err)
}

func main() {
	run()
}

func run() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9009"
	}
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = ioutil.Discard
	engine := gin.Default()
	config := cors.DefaultConfig()
	config.AllowOriginFunc = func(origin string) bool {
		return true
	}
	config.AllowCredentials = true
	config.AllowHeaders = []string{
		"Origin", "Content-Length", "Content-Type",
		"X-Screen-Height", "X-Screen-Width", "Authorization",
	}
	engine.Use(cors.New(config))

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", "mqtt.bamdad.dev", 1883))

	// opts.SetClientID("go_mqtt_client")
	// opts.SetUsername("emqx")
	// opts.SetPassword("public")
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	engine.Use(func(c *gin.Context) {
		c.Set("client", client)
	})

	app := engine.Group("/sandbox")
	api := app.Group("/api")

	api.POST("set/:color", func(c *gin.Context) {
		color := c.Param("color")
		cl, _ := c.Get("client")
		client := cl.(mqtt.Client)
		var rgb []string
		if color == "custom" {
			rgb = strings.Split(c.Query("rgb"), ",")
			if len(rgb) != 3 {
				c.JSON(http.StatusBadRequest, "not enough parameters")
				return
			}
			fmt.Println(rgb)
		}
		handleColor(client, color, rgb)

		c.JSON(http.StatusOK, "done")
		return
	})
	api.POST("/webhook", func(c *gin.Context) {
		fmt.Println("hook activated")
		var body WebHookReq
		err := c.ShouldBindJSON(&body)
		if err != nil {
			fmt.Println("webhook body error", err)
		}
		cl, _ := c.Get("client")
		client := cl.(mqtt.Client)
		rgb := []string{}
		if body.Color == "custom" {
			rgb = strings.Split(body.Action, ",")
		}
		handleColor(client, body.Color, rgb)
		c.JSON(http.StatusOK, "OK")
		fmt.Println("hook succeeded")
		return
	})

	fmt.Println(fmt.Println("LISTENING ON %s", port))
	endless.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", port), engine)
}

func publish(client mqtt.Client, topic, message string) {
	token := client.Publish(topic, 0, false, message)
	token.Wait()
}

func handleColor(client mqtt.Client, color string, rgb []string) {
	switch color {
	case "red":
		publish(client, "cmd/backlight1", "set/1020/0/0")
		break
	case "green":
		publish(client, "cmd/backlight1", "set/0/1020/0")
		break
	case "blue":
		publish(client, "cmd/backlight1", "set/0/0/1020")
		break
	case "purple":
		publish(client, "cmd/backlight1", "set/1000/0/800")
		break
	case "off":
		publish(client, "cmd/backlight1", "set/0/0/0")
		break
	case "custom":
		publish(client, "cmd/backlight1", fmt.Sprintf("set/%s/%s/%s", rgb[0], rgb[1], rgb[2]))
		break
	}
}
