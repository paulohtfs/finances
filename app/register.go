package main

import(
  "encoding/json"
  "encoding/csv"
  "fmt"
  "io/ioutil"
  "log"
  "net/http"
  "os"
  "strconv"
  "strings"

  "golang.org/x/net/context"
  "golang.org/x/oauth2"
  "golang.org/x/oauth2/google"
  "google.golang.org/api/sheets/v4"
)

var spreadsheetId = ""
var csvFile = ""

func getClient(config *oauth2.Config) *http.Client {
  tokFile := "token.json"
  tok, err := tokenFromFile(tokFile)
  if err != nil {
	tok = getTokenFromWeb(config)
	saveToken(tokFile, tok)
  }

  return config.Client(context.Background(), tok)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
  authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
  fmt.Printf("GO to the following link in your browser then type the authorization code: \n%v\n", authURL)

  var authCode string
  if _, err := fmt.Scan(&authCode); err != nil {
	log.Fatalf("Unable to read authorization code: %v", err)
  }

  tok, err := config.Exchange(context.TODO(), authCode)
  if err != nil {
	log.Fatalf("Unable to retrieve token from web: %v", err)
  }

  return tok
}

func tokenFromFile(file string) (*oauth2.Token, error) {
  f, err := os.Open(file)
  if err != nil {
	return nil, err
  }

  defer f.Close()
  tok := &oauth2.Token{}
  err = json.NewDecoder(f).Decode(tok)

  return tok, err
}

func saveToken(path string, token *oauth2.Token) {
  fmt.Printf("Saving credential file to: %s\n", path)
  f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
  if err != nil {
	log.Fatalf("Unable to cache oauth token: %v", err)
  }
  defer f.Close()
  json.NewEncoder(f).Encode(token)
}

func getConfig() (*oauth2.Config){
  b, err := ioutil.ReadFile("credentials.json")
  if err != nil {
	log.Fatalf("Unable to read client secret file: %v", err)
  }
  config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
  if err != nil {
	log.Fatalf("Unable to parse client secret file to config: %v", err)
  }

  return config
}

func getServer(client *http.Client) *sheets.Service{
  server, err := sheets.New(client)
  if err != nil {
	log.Fatalf("Unable to retrieve Sheets client: %v", err)
  }

  return server
}

func getValues(sheetRange string, server *sheets.Service) *sheets.ValueRange{
  resp, err := server.Spreadsheets.Values.Get(spreadsheetId, sheetRange).Do()
  if err != nil {
	log.Fatalf("Unable to retrieve data from sheet: %v", err)
  }

  return resp
}

// Gets tab, row and column dinamic
func getNext(server *sheets.Service) string{
  tab := "Transactions!"
  firstColumn := "B"
  firstLine := "5"
  lastColumn := "E"

  expensesRange := tab + firstColumn + firstLine + ":" + lastColumn
  expensesValueRange := getValues(expensesRange, server).Values
  totalRegisters := len(expensesValueRange)
  intFirstLine, _ := strconv.Atoi(firstLine)

  nextLine := totalRegisters + intFirstLine

  return tab + firstColumn + strconv.Itoa(nextLine)
}

func updateNext(current string) string{
  splitCurrent := strings.Split(current, "!")
  tab, xRange := splitCurrent[0], splitCurrent[1]
  column, line := xRange[0], xRange[1:]

  nextLine, _ := strconv.Atoi(line)
  nextLine += 1

  return tab + "!" + string(column) + strconv.Itoa(nextLine)
}

func writeLine(server *sheets.Service, line string, record []interface{}) {
  var writing sheets.ValueRange
  writing.Values = append(writing.Values, record)
 
  _, err := server.Spreadsheets.Values.Update(spreadsheetId, line, &writing).ValueInputOption("RAW").Do()
  if err != nil {
    log.Fatalf("Unable to write on sheet. %v", err)
  }
}

// CSV Methods
func rawCSV() string{
  data, err := ioutil.ReadFile(csvFile)
  if err != nil {
	panic(err)
  }

  return string(data)
}

func readCSV() [][]string{
  data := rawCSV()
  reader := csv.NewReader(strings.NewReader(data))

  records, err := reader.ReadAll()
  if err != nil {
	log.Fatal(err)
  }

  return records
}

func setArgs() {
  spreadsheetId = os.Args[1]
  csvFile = os.Args[2]
}

func main() {
  setArgs()

  config := getConfig()
  client := getClient(config)
  server := getServer(client)
  next := getNext(server)

  records := readCSV()
  records[0] = []string{""}

  fmt.Printf("%s \n", next)
  for _, record := range records {
	if len(record) == 4 {
	  recordInterface := []interface{}{record[0], record[3], record[2], record[1]}
	  writeLine(server, next, recordInterface)
	  next = updateNext(next)
	}
  }
}

