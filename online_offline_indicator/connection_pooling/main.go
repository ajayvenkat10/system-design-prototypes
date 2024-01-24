/*
	Lets say every user sends a heartbeat every 10 seconds. That will be 6 heartbeats per user per minute.
	So if you have a million users you get 6 million requests/minute for updating the heartbeat time. (Micro updates - update 1 row per call)
	Each micro update is still a DB query.

	Batching is one way to improve performance.
	What else can be done ?

	For each request, the server needs to establish a TCP connection with the database for each query.
	Lets say connection establishment and tear down (3 way handshake and 2 way tear down - TCP) takes 1 or 2 milliseconds.
	For a large query that takes around 50-100milliseconds or more, its not a problem but for micro updates that take within 2-3 milliseconds is a problem

	For example,
	Connection time/Large Update  - 1ms / 100ms
	Connection time/Micro Update  - 1ms / 2ms

	In teh above case you can see that connection time is 1% of query time in case of a large update which is not so large, but the connection time is
	is 50% of the query execution time in case of a micro update.

	To overcome this we use connection pooling.

	Connection pooling is a concept where you have a blocking queue of connections which you use from and the connection is put back into the
	queue for reusing after use.

	Concurrent requests will have to wait when no connection in the pool is available.

	This solves 2 problems:

	1. Remove extra time taken in establishing and tearing down the TCP connection
	2. DB server does not get overwhelmed when there is a spike in the nuber of concurrent connection requests.

	Connection pooling is offered as a part of configuring the DB for almost every DB server you host.
	Its an abstraction provided by the DB server to avoid such issues in production.

	Lets try to understand how it works under the hood with a small prototype.

	Lets take two approaches: Concurrent DB connections
		1. Without connection pooling
		2. With conection pooling

	We'll be using PostgreSQL as the Database.


	OUTPUT OF A DRY RUN:

	Without Connection Pooling:

	1. Benchmark: Without Connection Pool for  10 connections ->  52.8495ms
	2. Benchmark: Without Connection Pool for  100 connections ->  171.90475ms
	3. Benchmark: Without Connection Pool for  1000 connections ->  3.381637541s
	4. Benchmark: Without Connection Pool for  10000 connections:
		panic: failed to connect to `host=localhost user=postgres database=online_offline_indicator`: dial error (dial tcp 127.0.0.1:5432: connect: connection reset by peer)

		goroutine 4484 [running]:
		main.createConnectionAndSleep(0x0?)
				/Users/ajaymahadevan/Desktop/learning_go/prototypes/online_offline_indicator/connection_pooling/main.go:183 +0xe0
		created by main.benchmarkWithoutConnectionPool in goroutine 1
				/Users/ajaymahadevan/Desktop/learning_go/prototypes/online_offline_indicator/connection_pooling/main.go:204 +0x50
		exit status 2

		It threw an error for 10k requests. DB was overwhelmed and cannot accept that many concurrent requests.

	With Connection Pooling:

	1. Benchmark: With Connection Pool for  10 connections ->  53.543125ms
	2. Benchmark: With Connection Pool for  100 connections ->  147.729ms
	3. Benchmark: With Connection Pool for  1000 connections ->  1.163276125s
	4. Benchmark: With Connection Pool for  10000 connections ->  11.483939208s

	We can see that its not only faster with pooling , but also it did not throw an error for 10k requests and executed successfully.
	Thats because its not really concurrent here. We have a blocking wait for a new request when all the connections in the pool are in use and
	will proceed with a connection for this new request only when a connection is put back into the pool.

	Wait time increases but it does not throw an error.
*/

package main

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Connection struct {
	db *sql.DB
}

/*
	Mutex - for synchronization. The connection pool should be accessible only by one go routine at a time when you get or put.
	Channel - for buffer
	Connections - a slice containing total number of connection objects in the pool
	maximumConnections - Maximum capacity of the connection pool
*/

type ConnectionPool struct {
	mutexLock         *sync.Mutex
	channel           chan interface{}
	connections       []*Connection
	maximumConnection int
}

// Create a connection pool holding maxConnections
func newConnectionPool(maxConnections int) *ConnectionPool {
	pool := &ConnectionPool{
		mutexLock:         &sync.Mutex{},
		channel:           make(chan interface{}, maxConnections),
		connections:       make([]*Connection, 0, maxConnections),
		maximumConnection: maxConnections,
	}

	for i := 0; i < maxConnections; i++ {
		pool.connections = append(pool.connections, &Connection{db: newConnection()})
		pool.channel <- nil
	}

	return pool
}

// Establish connection with a postgres database
func newConnection() *sql.DB {
	_db, err := sql.Open("pgx", "host=localhost port=5432 dbname=online_offline_indicator user=postgres password=123wiki&*(")
	if err != nil {
		panic(err)
	}

	return _db
}

/*
Get connection from the blocking queue on pool and reduce the pool size by 1 by taking that connection out of the queue.
We implenent queue using a list. Remove from front (0th index) add from back (append).
*/
func (cPool *ConnectionPool) Get() *Connection {
	/*
		Emit from the channel. This emit will succeed if there is atleast one item in the channel.
		If not then the process will wait here until an item is put into the channel.
	*/
	<-cPool.channel

	cPool.mutexLock.Lock()
	conn := cPool.connections[0]
	cPool.connections = cPool.connections[1:]
	cPool.mutexLock.Unlock()

	return conn
}

/*
Put the connection back into the pool after use..
*/
func (cPool *ConnectionPool) Put(conn *Connection) {
	cPool.mutexLock.Lock()
	cPool.connections = append(cPool.connections, conn)
	cPool.mutexLock.Unlock()

	// Put an item into the channel to indicate a connection is available
	cPool.channel <- nil
}

// Closing all the connections in the pool
func (cPool *ConnectionPool) Close() {
	close(cPool.channel)
	for i := range cPool.connections {
		cPool.connections[i].db.Close()
	}
}

// Can perform any query. Just calling sleep for 10 seconds for each connection.
func getConnectionFromPoolAndSleep(wg *sync.WaitGroup, pool *ConnectionPool) {

	defer wg.Done()

	connection := pool.Get()
	query := `SELECT pg_sleep(0.01)`
	_, err := connection.db.Exec(query)
	if err != nil {
		panic(err)
	}

	pool.Put(connection)
}

// Benchmarking without connection pooling
func benchmarkWithConnectionPool(concurrentConnections int) {
	startTime := time.Now()
	pool := newConnectionPool(10)

	var wg sync.WaitGroup

	for i := 0; i < concurrentConnections; i++ {
		wg.Add(1)
		go getConnectionFromPoolAndSleep(&wg, pool)
	}

	wg.Wait()
	pool.Close()

	fmt.Println("Benchmark: With Connection Pool for ", concurrentConnections, "connections -> ", time.Since(startTime))
}

// Can perform any query. Just calling sleep for 10 seconds for each connection.
func createConnectionAndSleep(wg *sync.WaitGroup) {
	defer wg.Done()

	db := newConnection()
	query := `SELECT pg_sleep(0.01)`
	_, err := db.Exec(query)
	if err != nil {
		panic(err)
	}
	db.Close()
}

// Benchmarking without connection pooling
func benchmarkWithoutConnectionPool(concurrentConnections int) {
	startTime := time.Now()

	/*
		Using WaitGroups for synchronization of threads.
		1. Before starting a goroutine, you increment the WaitGroup counter using the Add method. This informs the WaitGroup that a goroutine is about to start.
		3. Once the goroutine is completed, you need to call the Done() to signal the completion. This will decrement the counter behind the scenes
		4. Outside of this you call the Wait() which is similar to the await. Your program waits for the goroutines in the waitgroup to complete execution.
		5. To understand better, wait is called on the waitgroup and you program waits there until all go routines in the waitgroup have called Done() and the counter is 0.
	*/
	var wg sync.WaitGroup

	for i := 0; i < concurrentConnections; i++ {

		wg.Add(1)
		go createConnectionAndSleep(&wg)
	}

	wg.Wait()

	fmt.Println("Benchmark: Without Connection Pool for ", concurrentConnections, "connections -> ", time.Since(startTime))
}

func main() {

	n := 10

	benchmarkWithoutConnectionPool(n)
	benchmarkWithConnectionPool(n)
}
