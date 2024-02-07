package main

import (
	"database/sql"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

const HEARTBEAT_TIMEOUT_SECONDS = 30

var DBs []*sql.DB

func init() {
	// For eg "root:abcd@tcp(localhost:3306)/online_offline_indicator"
	_dB, err := sql.Open("mysql", os.Getenv("SQL_DB_DATA"))
	if err != nil {
		panic(err)
	}
	DBs = append(DBs, _dB)

	_dB, err = sql.Open("mysql", os.Getenv("SQL_DB_DATA"))
	if err != nil {
		panic(err)
	}
	DBs = append(DBs, _dB)
}

func getShardIndex(userId string) int {
	userId = strings.TrimSpace(userId)
	userIdInt, err := strconv.Atoi(userId)

	if err != nil {
		panic(err)
	}

	return userIdInt % len(DBs)
}

func isHeartbeatTimerActive(lastHeartbeatEpochMilliseconds int) bool {
	return (int(time.Now().Unix()) - lastHeartbeatEpochMilliseconds) <= HEARTBEAT_TIMEOUT_SECONDS
}

func main() {

	ge := gin.Default()

	ge.POST("/heartbeats", func(ctx *gin.Context) {
		request := make(map[string]interface{})
		ctx.Bind(&request)

		query := "REPLACE INTO online_offline (user_id, last_heartbeat) VALUES (?, ?);"

		DB := DBs[getShardIndex(strconv.Itoa(int(request["user_id"].(float64))))]

		_, err := DB.Exec(query, request["user_id"], time.Now().Unix())
		if err != nil {
			panic(err)
		}

		ctx.JSON(200, map[string]interface{}{"message": "ok"})
	})

	ge.GET("/heartbeats/status/:user_id", func(ctx *gin.Context) {
		var lastHeartbeat int

		query := "SELECT last_heartbeat FROM online_offline WHERE user_id = ?;"

		DB := DBs[getShardIndex(ctx.Param("user_id"))]

		row := DB.QueryRow(query, ctx.Param("user_id"))
		row.Scan(&lastHeartbeat)

		ctx.JSON(200, map[string]interface{}{"isOnline": isHeartbeatTimerActive(lastHeartbeat)})

	})

	ge.Run(":9000")

}
