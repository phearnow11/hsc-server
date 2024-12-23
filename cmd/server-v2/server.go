package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"pigdata/datadog/processingServer/internal/metric"
	"pigdata/datadog/processingServer/pkg/utils"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
)

// Use cronjob to save data to database, request get data from dd
var (
	app                  = fiber.New(fiber.Config{})
	redisStatus    int32 = 0
	location             = &time.Location{}
	base_line_data       = make(map[string]interface{})
)

func init() {
	godotenv.Load(".env")
	location, _ = time.LoadLocation("Asia/Bangkok") // Commonly used for GMT+7
}

func main() {
	osname := runtime.GOOS
	// a
	log.Println("Starting process", os.Getpid(), "on", osname)
	log_dir := ""
	if osname == "windows" {

		cur_dir, err := os.Getwd()
		if err != nil {
			log.Println("Error getting directory:", err)
		}
		log_dir = cur_dir + "\\log\\"
	} else if osname == "linux" {
		log_dir = "./log/"
	} else {
		log.Fatal("System", osname, "is not supported")
	}

	cur_time := time.Now().In(location)
	if cur_time.Second() < 3 {
		cur_time.Add(-1 * time.Minute)
	}

	log_name := log_dir + "log_server_at_" + cur_time.Format("2006-02-01:15-04") + ".log"
	logDir := filepath.Dir(log_name)

	err := utils.CreateFile(logDir)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.OpenFile(log_name, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,DELETE",
		AllowHeaders:     "*",
		AllowCredentials: false,
	}))

	apiKey := os.Getenv("PS_API_KEY")
	Metric := app.Group("api/v2/metrics", Middleware(apiKey))
	Metric.Post("/:metric_type/:cluster", MetricHandler())
	Metric.Get("business/baseline", ChartBaseline())
	Status := app.Group("api/v2/status", Middleware(apiKey))
	Status.Post("/:status_type/:cluster", StatusHandler())
	MetricDetail := app.Group("api/v2/metrics/detail", Middleware(apiKey))
	MetricDetail.Get("/:cluster/:name", ServiceMetricDetailHandler())
	app.Post("/webhook/order-update", WebhookHandlerOrderUpdate())
	app.Post("/webhook/save", WebhookHandler())
	Admin := app.Group("api/v2/admin", Middleware(apiKey))
	Admin.Post("reset/query", func(c *fiber.Ctx) error {
		return utils.Reset_query()
	})
	Admin.Post("/trigger-log/datadog-data", func(c *fiber.Ctx) error {
		query := c.Query("query")
		str_interval := c.Query("time")
		interval, err := strconv.Atoi(str_interval)
		if err != nil {
			return c.JSON(err)
		}
		to := time.Now().In(location)
		from := to.Add(time.Minute * -time.Duration(interval))
		data := metric.TimeseriesPointQueryData(from.Unix(), to.Unix(), query)
		trigger_log([]interface{}{to_string_datadog(data)})

		return c.JSON(data)
	})
	Admin.Post("/trigger-log/blocked-ip", func(c *fiber.Ctx) error {
		log_blocked_ip_msg := "Current Blocked IP status:"
		trigger_log([]interface{}{log_blocked_ip_msg, blacklistIP})
		return c.SendString("Log Triggered, check detail in log file")
	})
	Admin.Post("/block/:ip", func(c *fiber.Ctx) error {
		ip := c.Params("ip")
		blacklistIP[ip] = true
		log.Println("BLOCK IP REQUEST: IP", ip, "by:", c.IP())
		return c.SendString("Blocked " + ip)
	})
	Admin.Post("/unblock/:ip", func(c *fiber.Ctx) error {
		ip := c.Params("ip")
		blacklistIP[ip] = false
		log.Println("UNBLOCK IP REQUEST: IP", ip, "by:", c.IP())
		return c.SendString("Blocked " + ip)
	})

	server_port := os.Getenv("SERVER_PORT")

	// mongodb.Init_database()
	go func() {
		for {
			queries := utils.Metrics
			to := time.Now().In(location)
			from := to.Add(time.Minute * -30)
			for service_name, query := range queries.ServiceMetricDetailServiceSuccessRate {
				ServiceMetricDetailSuccessRateService(service_name, query, from.Unix(), to.Unix())
			}
			time.Sleep(time.Minute * 10)
		}
	}()

	go func() {
		BaseLineCal()
		cur_date := time.Now()
		first_call := time.Date(cur_date.Year(), cur_date.Month(), cur_date.Day(), 9, 30, 0, 0, location)
		second_call := time.Date(cur_date.Year(), cur_date.Month(), cur_date.Day(), 16, 0, 0, 0, location)
		// aaa
		// from_test := time.Date(cur_date.Year(), cur_date.Month(), cur_date.Day(), 9, 45, 0, 0, location)
		// to_test := time.Date(cur_date.Year(), cur_date.Month(), cur_date.Day(), 9, 48, 0, 0, location)

		for {
			if cur_date.Hour() == first_call.Hour() || cur_date.Hour() == second_call.Hour() {
				BaseLineCal()
			}
			queries := utils.Metrics
			status_data_msg := "SCHEDULE GET STATUS DATA:"
			to := time.Now().In(location)
			from := to.Add(time.Minute * -6)

			for _, cluster := range queries.ServiceMetric {
				for name, dashboard := range cluster {
					status_data_new[name] = GetStatusData(name, dashboard["Avail"], from.Unix(), to.Unix())
					log.Println(status_data_msg, "Service Metric Available status:", status_data_new[name], "for:", name)
				}
			}

			for service_name, query := range queries.ErrorDetailServiceMetricDetail {
				ServiceMetricErrorDetail(service_name, query, from.Unix(), to.Unix())
			}
			for service_name, query := range queries.ServiceMetricDetailSuccessRate {
				ServiceMetricDetailSuccessRateTestName(service_name, query, from.Unix(), to.Unix())
			}

			time.Sleep(time.Minute * 1)
		}
	}()

	// Start cronjob
	if script_status {
		go func() {
			for {
				runScripts(scripts_config.Scripts)
				time.Sleep(_interval)
			}
		}()
	}

	// Start Server
	go func() {
		log.Fatal(app.Listen(":" + server_port))
	}()

	if strings.ToUpper(osname) == "LINUX" {
		sigChannel := make(chan os.Signal, 1)
		signal.Notify(sigChannel, syscall.SIGINT, syscall.SIGTERM)

		sig := <-sigChannel
		log.Println("Received signal:", sig)

		log.Println("Server shut down gracefully")
	}
}
