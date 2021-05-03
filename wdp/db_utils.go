// implements the backend, often-used pieces of the database functionality for
// the web app

package wdp

import (
	"database/sql"
	"context"
	"log"
	"time"
	"fmt"
	"strings"
    "os"
    "bufio"
    "encoding/csv"
    "strconv"
    "math/rand"

    _ "github.com/go-sql-driver/mysql"

)

// for the creation of UIDS 
var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// gets the DSN based on an input string
func DSN(dbName string) (string, error) {
    // NOTE: SWITCH WHICH OF THESE STATEMENTS IS COMMENTED OUT TO RUN ON CLOUD VPS VS LOCALLY
    // f, err := os.Open("/Users/haltriedman/replica.my.cnf") // LOCAL
    f, err := os.Open("/home/htriedman/replica.my.cnf") // CLOUD VPS
    defer f.Close()
    if err != nil {
        fmt.Printf("Error %s when opening replica file", err)
        return "", err
    }

    scanner := bufio.NewScanner(f)

    scanner.Split(bufio.ScanLines)
    var username string
    var password string
  
    for scanner.Scan() {
        str_split := strings.Split(scanner.Text(), " = ")
        if str_split[0] == "user" {
            username = str_split[1]
        } else if str_split[0] == "password" {
            password = str_split[1]
        }
    }

    // return DSN
    return fmt.Sprintf("%s:%s@tcp(127.0.0.1)/%s", username, password, dbName), nil
}

// creates the DB if it doesn't already exist, and returns a connection to the DB
func DBConnection() (*sql.DB, error) {
    // PART 1: CREATE DB IF IT DOESN'T ALREADY EXIST

    // get DSN
    dbName, err := DSN("")
    if err != nil {
    	log.Printf("Error %s when getting dbname\n", err)
    	return nil, err
    }

    // open DB
    db, err := sql.Open("mysql", dbName)
    if err != nil {
        log.Printf("Error %s when opening DB\n", err)
        return nil, err
    }

    // set context
    ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancelfunc()

    // create DB if not exists
    res, err := db.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS wdp")
    if err != nil {
        log.Printf("Error %s when creating DB\n", err)
        return nil, err
    }

    // see how many rows affected (should be 0)
    no, err := res.RowsAffected()
    if err != nil {
        log.Printf("Error %s when fetching rows", err)
        return nil, err
    }
    log.Printf("rows affected %d\n", no)

    db.Close()

    // PART 2: CONNECT TO EXISTING DB

    // get DSN again, this time for the specific DB we just made
    dbName, err = DSN("wdp")
    if err != nil {
    	log.Printf("Error %s when getting dbname\n", err)
    	return nil, err
    }

    // open the DB
    db, err = sql.Open("mysql", dbName)
    if err != nil {
        log.Printf("Error %s when opening DB", err)
        return nil, err
    }

    // config stuff — TODO: this might have to go
    // db.SetMaxOpenConns(60)
    // db.SetMaxIdleConns(30)
    db.SetMaxIdleConns(0)


    // set context
    ctx, cancelfunc = context.WithTimeout(context.Background(), 5*time.Second)
    defer cancelfunc()

    // make sure connection works by pinging
    err = db.PingContext(ctx)
    if err != nil {
        log.Printf("Errors %s pinging DB", err)
        return nil, err
    }

    log.Printf("Connected to DB successfully\n")
    return db, nil
}

// creates a table with name tbl_name in DB db. Called in init_db.go.
func CreateTable(db *sql.DB, tbl_name string) error {
    var query string

    // check to make sure tbl_name is right format, then construct query based on type
    if tbl_name == "data" {
        query = `CREATE TABLE IF NOT EXISTS data(pv_id INT PRIMARY KEY AUTO_INCREMENT, user_id TEXT, day DATE, lang TEXT, name TEXT)`
    } else if tbl_name == "output" {
        query = `CREATE TABLE IF NOT EXISTS output(Name TEXT, Views INT, Lang TEXT, Day DATE, Kind TEXT, Epsilon FLOAT, Delta FLOAT, Sensitivity INT)`
    } else {
        return fmt.Errorf("input to create table was not properly formated: %s", tbl_name)
    }

    // set context
    ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancelfunc()

    // execute query and check rows affected (should be 0)
    res, err := db.ExecContext(ctx, query)
    if err != nil {
        log.Printf("Error %s when creating product table", err)
        return err
    }
    rows, err := res.RowsAffected()
    if err != nil {
        log.Printf("Error %s when getting rows affected", err)
        return err
    }

    log.Printf("Rows affected when creating table: %d\n", rows)
    return nil
}

// function for inserting faux data for a specific language in batches so as not
// to overwhelm the limits of mysql for loading data (which is around ~50,000 placeholders)
func BatchInsert(db *sql.DB, date, lang string, topFiftyArticles [50]Article) error {
    // initialize things to insert, batch counter, and query string
    var inserts []string
    var params []interface{}
    batch := 0
    query := "INSERT INTO data (user_id, day, lang, name) VALUES "
    viewsize := 0
    for i := 0; i < 50; i++ {
        viewsize += topFiftyArticles[i].Views
    }

    wikisize, ok := LanguageMap[lang]
    if !ok {
        return fmt.Errorf("Language to insert is not in LanguageMap in validate.go")
    }

    uids, err := initUsers(wikisize, viewsize)
    if err != nil {
        log.Printf("Error initializing UIDs %s", err)
        return err
    }

    uidCounter := 0
    // for each of the top fifty articles
    for i := 0; i < 50; i++ {
        // for the number of views that it has
        for j := 0; j < topFiftyArticles[i].Views; j++ {
            // append a parameterized variable to the query and the name of the page
            inserts = append(inserts, "(?, ?, ?, ?)")
            params = append(params, uids[uidCounter], date, lang, topFiftyArticles[i].Name)

            // increment the batch counter and the UID counter
            batch++
            uidCounter++

            // if the batch counter is 10,000 or greater
            if batch >= 10000 {

                // insert the values into the db
                err := insert(db, query, inserts, params)
                if err != nil {
                    log.Printf("error %s while inserting into data table", err)
                }

                // reset everything back to 0/empty list
                inserts = nil
                params = nil
                batch = 0
            }
        }
    }

    // insert whatever is left at the end
    err = insert(db, query, inserts, params)
    if err != nil {
        log.Printf("error %s while inserting", err)
    }

    return nil
}



// The actual workhorse of the inserion process. Safely inserts a set of params
// into a query and adds the whole thing to the database.
func insert(db *sql.DB, query string, inserts []string, params []interface{}) error {
    // create the query based on the insert list
    queryVals := strings.Join(inserts, ",")
    query = query + queryVals

    // set context
    ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancelfunc()

    // prepare the query
    stmt, err := db.PrepareContext(ctx, query)
    if err != nil {
        log.Printf("Error %s when preparing SQL statement", err)
        return err
    }
    defer stmt.Close()

    // execute the query and see how many rows were affected
    res, err := stmt.ExecContext(ctx, params...)
    if err != nil {
        log.Printf("Error %s when inserting row into table", err)
        return err
    }
    rows, err := res.RowsAffected()
    if err != nil {
        log.Printf("Error %s when finding rows affected", err)
        return err
    }
    log.Printf("%d sessions created ", rows)
    return nil
}

// function to query the database and get back the normal count and a DP count
// based on the input (epsilon, delta) tuple
func Query(db *sql.DB, lang, kind string, epsilon, delta float64, sensitivity int) ([]TableRow, []TableRow, error) {
    // set up output structs
    var normalCount []TableRow
    var dpCount []TableRow

    // create the query -- use the mysql round function to get around the fact that floats are imprecise
    // the inner join filters to just the most recent day of data, and lang filters the language
    // -1 is the code for normal, so we get -1 and the input epsilon and delta
    var query = `
        SELECT Name, Views, Lang, Day, Kind, Epsilon, Delta, Sensitivity FROM output
        INNER JOIN (
            SELECT MAX(Day) as max_day
            FROM output
        ) sub
        ON output.Day=sub.max_day
        WHERE
            ((Epsilon=-1
            AND Delta=-1)
            OR
            (ROUND(Epsilon, 1)=ROUND(?, 1)
            AND ROUND(Delta, 9)=ROUND(?, 9)
            AND Kind=?
            AND Sensitivity=?))
            AND Lang=?
        `

    // set context
    ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancelfunc()

    // query the table
    res, err := db.QueryContext(ctx, query, epsilon, delta, kind, sensitivity, lang)
    if err != nil {
        log.Printf("Error %s when conducting query", err)
        return normalCount, dpCount, err
    }
    defer res.Close()
    

    // iterate through results
    for res.Next() {
        var row TableRow

        res.Scan(&row.Name, &row.Views, &row.Lang, &row.Day, &row.Kind, &row.Epsilon, &row.Delta, &row.Sensitivity)

        // if epsilon or delta is -1, add to the normal list; otherwise, add to the dpcount list
        if row.Epsilon == float64(-1) || row.Delta == float64(-1) {
            normalCount = append(normalCount, row)
        } else {
            dpCount = append(dpCount, row)
        }
    }

    return normalCount, dpCount, nil
}

// function for systematically dropping the rows of old data from previous days.
// called in clean_db.go, and should be used after beam.go does aggregating.
func DropOldData(db *sql.DB, tbl_name, date string) error {
    if tbl_name != "data" && tbl_name != "output" {
        return fmt.Errorf("Error: incorrect formatting for table name %s", tbl_name)
    }

    query := `DELETE FROM ` + tbl_name + ` WHERE day != "` + date + `"`

    // set context
    ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancelfunc()

    // execute the row deletion
    res, err := db.ExecContext(ctx, query)
    if err != nil {
        log.Printf("Error %s deleting rows", err)
        return err
    }

    rows, err := res.RowsAffected()
    if err != nil {
        log.Printf("Error %s when finding rows affected", err)
        return err
    }
    log.Printf("%d pageviews deleted ", rows)
    return nil
}


// initializes a list of randomly ordered userhashes that occur with the frequency
// given in {large, medium, small}wiki.csv.
// Outputs a list of length viewsize
func initUsers(wikisize string, viewsize int) ([]string, error) {
    var output []string

    // read in the size of the wiki as a csv
    // NOTE: SWITCH THESE TO RUN LOCALLY VS CLOUD VPS
    // f, err := os.Open(fmt.Sprintf("data/%swikis.csv", wikisize)) // LOCAL
    f, err := os.Open(fmt.Sprintf("/etc/diff-privacy-beam/%swikis.csv", wikisize)) // CLOUD VPS
    if err != nil {
        log.Printf("Error %s opening csv", err)
        return output, err
    }
    r := csv.NewReader(f)
    records, err := r.ReadAll()
    if err != nil {
        log.Printf("Error %s reading csv", err)
        return output, err
    }

    // for each row in the pdf
    for _, row := range records {
        // convert num views to an int
        numViews, err := strconv.Atoi(row[0])
        if err != nil {
            log.Printf("Error %s converting to int", err)
            return output, err
        }
        // convert proportion to a float
        proportion, err := strconv.ParseFloat(row[1], 64)
        if err != nil {
            log.Printf("Error %s converting to float", err)
            return output, err
        }
        // get estimated number of users with that many views
        estNumUsers := int((proportion * float64(viewsize)) / float64(numViews))

        // for each user, generate an id and append it numViews many times to the output
        for i := 0; i < estNumUsers; i++ {
            uid := generateUID(10) // assume no collisions for the moment
            for j := 0; j < numViews; j++ {
                output = append(output, uid)
            }
        }
    }

    // shuffle order of views so that they will not all fall in one kind of page
    rand.Seed(time.Now().Unix())
    rand.Shuffle(len(output), func(i, j int) {
        output[i], output[j] = output[j], output[i]
    })

    // ensure that output is the right length (because this process involves rounding)

    // if too short, generate more single-view people
    for len(output) < viewsize {
        uid := generateUID(10)
        output = append(output, uid)
    }

    // if too long, cut to the correct size
    if len(output) > viewsize {
        output = output[:viewsize]
    }
    return output, nil
}

// returns a randomly generated n-character UID
func generateUID(n int) string {
    rand.Seed(time.Now().Unix())
    b := make([]rune, n)
    for i := range b {
        b[i] = letters[rand.Intn(len(letters))]
    }
    return string(b)
}
