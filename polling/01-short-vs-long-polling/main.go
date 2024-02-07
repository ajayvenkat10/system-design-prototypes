package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func init() {
	// For eg: root:abc@tcp(localhost:3306)/short_long_polling
	_db, err := sql.Open("mysql", os.Getenv("DB_DATA"))
	if err != nil {
		panic(err)
	}

	DB = _db
}

func createEC2(serverId int) {
	fmt.Println("Creating server...")

	_, err := DB.Exec("UPDATE p_servers SET status = 'TODO' WHERE server_id = ?;", serverId)
	if err != nil {
		panic(err)
	}

	time.Sleep(5 * time.Second)
	_, err = DB.Exec("UPDATE p_servers SET status = 'IN PROGRESS' WHERE server_id = ?;", serverId)
	if err != nil {
		panic(err)
	}

	fmt.Println("Server creation in Progress...")

	time.Sleep(5 * time.Second)
	_, err = DB.Exec("UPDATE p_servers SET status = 'DONE' WHERE server_id = ?;", serverId)
	if err != nil {
		panic(err)
	}

	fmt.Println("Server creation is Done...")
}

func main() {
	ge := gin.Default()

	ge.POST("/servers", func(ctx *gin.Context) {
		request := make(map[string]interface{})
		ctx.Bind(&request)

		serverId := int(request["server_id"].(float64))

		go createEC2(serverId)

		ctx.JSON(200, map[string]interface{}{"status": "ok"})
	})

	ge.GET("/short/status/:server_id", func(ctx *gin.Context) {
		serverId := ctx.Param("server_id")

		var status string

		row := DB.QueryRow("SELECT status FROM p_servers WHERE server_id = ?;", serverId)

		if row.Err() != nil {
			panic(row.Err())
		}

		row.Scan(&status)

		ctx.JSON(200, map[string]interface{}{"status": status})
	})

	ge.GET("/long/status/:server_id", func(ctx *gin.Context) {
		serverId := ctx.Param("server_id")
		currentStatus := ctx.Query("currentStatus")

		var status string

		for {
			row := DB.QueryRow("SELECT status FROM p_servers WHERE server_id = ?;", serverId)

			if row.Err() != nil {
				panic(row.Err())
			}

			row.Scan(&status)

			if status != currentStatus {
				break
			}
		}

		ctx.JSON(200, map[string]interface{}{"status": status})
	})

	ge.Run(":9000")
}
