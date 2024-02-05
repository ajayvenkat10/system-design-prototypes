package main

import (
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const HEARTBEAT_TIMEOUT_SECONDS = 30

var DBs []*sql.DB

func init() {
	_dB, err := sql.Open("pgx", "host=localhost port=5432 dbname=online_offline_indicator user=postgres password=123wiki&*(")
	if err != nil {
		panic(err)
	}
	DBs = append(DBs, _dB)

	_dB, err = sql.Open("pgx", "host=localhost port=5432 dbname=online_offline_indicator_slave user=postgres password=123wiki&*(")
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

		query := `INSERT INTO online_offline (user_id, last_heartbeat) VALUES ($1, $2) ON CONFLICT (user_id) DO UPDATE SET last_heartbeat = EXCLUDED.last_heartbeat`

		DB := DBs[getShardIndex(strconv.Itoa(int(request["user_id"].(float64))))]

		_, err := DB.Exec(query, request["user_id"], time.Now().Unix())
		if err != nil {
			panic(err)
		}

		ctx.JSON(200, map[string]interface{}{"message": "ok"})
	})

	ge.GET("/heartbeats/status/:user_id", func(ctx *gin.Context) {
		var lastHeartbeat int

		query := `SELECT last_heartbeat FROM online_offline WHERE user_id = $1`

		DB := DBs[getShardIndex(ctx.Param("user_id"))]

		row := DB.QueryRow(query, ctx.Param("user_id"))
		row.Scan(&lastHeartbeat)

		ctx.JSON(200, map[string]interface{}{"isOnline": isHeartbeatTimerActive(lastHeartbeat)})

	})

	ge.Run(":9000")

}
