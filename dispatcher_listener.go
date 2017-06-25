package main

import (
    "fmt"
    "log"

    "encoding/json"
    "time"

    "database/sql"
    "github.com/lib/pq"
)



// read all tasks from the database
func (d *Dispatcher) firstRead() {

    // retrieve all task
    var tasks []Task

    _, err := d.connector.Select(&tasks, "select * from task where status = :status order by id asc", map[string]interface{}{"status":"todo"})

    if err != nil {
        log.Fatalln("Select failed", err)
    }

    log.Println("All rows:")

    for x, task := range tasks {
        TaskQueue <- task

        log.Printf("  %d : %v\n", x, task)
    }
}




// prepare the listener data and launch it
func (d *Dispatcher) initializeListenerAndLaunch() {

    _, err := sql.Open("postgres", ConnectionConfiguration)

    if err != nil {
        panic(err)
    }

    reportProblem := func(ev pq.ListenerEventType, err error) {
        if err != nil {
            fmt.Println(err.Error())
        }
    }

    listener := pq.NewListener(ConnectionConfiguration, 10*time.Second, time.Minute, reportProblem)

    err = listener.Listen("events_task")

    if err != nil {
        panic(err)
    }

    fmt.Println("Start monitoring PostgreSQL...")

    for {
        d.waitForNotificationFromListener(listener)
    }
}


type DatabaseNotification struct {
    Table   string
    Action  string
    Task    Task
}

// listening to the event bus of the database and do some actions
func (d *Dispatcher) waitForNotificationFromListener(l *pq.Listener) {
    for {
        select {
            case n := <-l.Notify:
                var notification DatabaseNotification

                err := json.Unmarshal([]byte(n.Extra), &notification)

                if err != nil {
                    fmt.Println("error:",err)
                }

                var task = notification.Task

                if task.Status != "todo" {
                    return
                }

                fmt.Println("Received data from channel [", n.Channel, "] :")

                fmt.Printf("%+v \n", notification)

                TaskQueue <- notification.Task

                return

            case <-time.After(90 * time.Second):
                fmt.Println("Received no events for 90 seconds, checking connection")

                go func() {
                    l.Ping()
                }()

                // retrieve all task
                var tasks []Task

                _, err := d.connector.Select(&tasks, "select * from task where status = :status order by id asc", map[string]interface{}{"status":"todo"})

                if err != nil {
                    log.Fatalln("Select failed", err)
                }

                log.Println("All rows:")

                for x, task := range tasks {
                    TaskQueue <- task

                    log.Printf("  %d : %v\n", x, task)
                }

                return
        }
    }
}