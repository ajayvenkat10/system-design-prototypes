/*

Designing Online Offline Indicator.

- What is online offline indicator ??
Its to indiacte whether a user is online or offline. Use case : Social Media, Communication Apps etc

	Two approaches : PUSH and PULL

	1. PUSH - Client/User pushes their status to the server with an API call periodically
	2. PULL - API server fetches the status periodically from the user/client (Would involve biderectional communication : Web Sockets, Server Sent Events etc )

	We'll be exploring thex PUSH based approach in this program.

Hunch for approach 1:

	1. User has to periodically make an API call to update the server with their online status. Lets call this API the heartbeat API
	2. Lets say we are working with SQL. How would our DB table schema look like ? (user_id int, is_online bool) ? Not a bad start
	3. Whenever ther user calls the heartbeat API (POST endpoint) we'll update the DB entry for is_online with true.
	4. When will we set it to false ? Maybe when the user logs out ?
	5. Is that the best way though ? Most social media and communication apps don't need a logout. Their sessions are refreshed and alive for
		significantly long duration. User not online means that the user has closed the application on their device or browser. Also in case of a crash,
		we can't guarantee that the logout API will be called as well. So how can we improve this ?
	6. We can store the timestamp of each heartbeat and when there is a GET API call to fetch the status, we can check if enough time has passed since the
		user called the heartbeat API or not to dertermine if the user is online or offline.

Approach: (Lets say we are working with an SQL Database)

	1. User/Client will periodically call an API to update their status- lets call this the heartbeat API (POST endpoint)
	2. When the above API is called, we write the new heartbeat time to DB. Now the DB schema would look like: (user_id int, last_heartbeat int).
		We could use the unix epoch milliseconds instead of timestamp for easy calculation and comparison for our use case.
	3. There will be GET endpoint/endpoints to fetch the status of a single or a batch of users.
	4. For a single user, its just a straightforward DB read call.
	5. For n users, we use the batch API.

	We'll be seeig 2 such batching approaches and understand which is better.
	Case 1 : Non optimal - We fire a DB query for each user in the list of users obtained in teh request to get their status.
	Case 2 : Optimal - We fira a single DB query to fetch the status of the the list of users .

	The DB used in this prototype is **PostgreSQL**.

	Output of a dry run:

		2024/01/21 - 13:52:53 | 200 | 3.20825ms | 127.0.0.1 | GET "/heartbeats/status_non_optimized?user_ids=1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20"
		2024/01/21 - 13:53:00 | 200 | 609.459µs | 127.0.0.1 | GET "/heartbeats/status_optimized?user_ids=1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20"


	You can see from the dry run above that the non optimal way took about 3.2 milliseconds to get the status of 20 users.
	Whereas the optimal way took about 609.5 micro seconds to get the status of 20 users.

	Takeaways:

	1. Reduce the number of API calls by batching wherever possible. Saves time and reduces load on your API server.
	2. Batch efficiently by reducing the number of read/write calls to your database to reduce the load on the database and in turn improve the speed of response.
		Try to optimize for batch reads/writes than a read/write for every user/request

*/

package main

import (
	"database/sql"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const HEARTBEAT_TIMEOUT_SECONDS = 30

var DB *sql.DB

func init() {
	// Example DB Data: "host=localhost port=5432 dbname=online_offline_indicator user=postgres password=postgres123")
	_db, err := sql.Open("pgx", os.Getenv("DB_DATA"))
	if err != nil {
		panic(err)
	}

	DB = _db
}

func isHeartbeatTimerActive(lastHeartbeatEpochMilliseconds int) bool {
	return (int(time.Now().Unix()) - lastHeartbeatEpochMilliseconds) <= HEARTBEAT_TIMEOUT_SECONDS
}

func main() {
	// Using the gin library which makes it easy to build REST APIs and server
	ge := gin.Default()

	ge.POST("/heartbeats", func(ctx *gin.Context) {
		data := make(map[string]interface{})
		ctx.Bind(&data)

		query := `INSERT INTO online_offline (user_id, last_heartbeat) VALUES ($1, $2) ON CONFLICT (user_id) DO UPDATE SET last_heartbeat = EXCLUDED.last_heartbeat`

		_, err := DB.Exec(query, data["user_id"], time.Now().Unix())
		if err != nil {
			panic(err)
		}

		ctx.JSON(200, map[string]interface{}{"message": "ok"})
	})

	ge.GET("/heartbeats/status/:user_id", func(ctx *gin.Context) {
		var lastHeartbeat int
		query := `SELECT last_heartbeat FROM online_offline WHERE user_id = $1`

		row := DB.QueryRow(query, ctx.Param("user_id"))
		row.Scan(&lastHeartbeat)

		isOnline := isHeartbeatTimerActive(lastHeartbeat)

		ctx.JSON(200, map[string]interface{}{"is_online": isOnline})
	})

	ge.GET("heartbeats/status_non_optimized", func(ctx *gin.Context) {
		activeStatusMap := make(map[string]bool)
		query := `SELECT last_heartbeat FROM online_offline WHERE user_id = $1`

		for _, userID := range strings.Split(ctx.Query("user_ids"), ",") {
			var lastHeartbeat int
			row := DB.QueryRow(query, userID)
			err := row.Scan(&lastHeartbeat)
			if err != nil {
				panic(err)
			}
			activeStatusMap[userID] = isHeartbeatTimerActive(lastHeartbeat)
		}

		ctx.JSON(200, activeStatusMap)

	})

	ge.GET("heartbeats/status_optimized", func(ctx *gin.Context) {
		activeStatusMap := make(map[string]bool)

		query := `SELECT user_id, last_heartbeat FROM online_offline WHERE user_id IN (` + ctx.Query("user_ids") + `)`

		rows, err := DB.Query(query)
		if err != nil {
			panic(err)
		}

		var userID, lastHeartbeat int
		for rows.Next() {
			err := rows.Scan(&userID, &lastHeartbeat)
			if err != nil {
				panic(err)
			}

			activeStatusMap[strconv.Itoa(userID)] = isHeartbeatTimerActive(lastHeartbeat)
		}
		rows.Close()

		ctx.JSON(200, activeStatusMap)
	})

	ge.Run(":9000")
}
