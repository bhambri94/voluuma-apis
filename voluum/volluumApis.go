package voluum

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bhambri94/voluum-apis/configs"
	config "github.com/bhambri94/voluum-apis/configs"
)

type AuthApiResponse struct {
	Token               string    `json:"token"`
	ExpirationTimestamp time.Time `json:"expirationTimestamp"`
	Inaugural           bool      `json:"inaugural"`
}

type AuthApiRequest struct {
	AccessID  string `json:"accessId"`
	AccessKey string `json:"accessKey"`
}

type DailyReport struct {
	TotalRows int `json:"totalRows"`
	Rows      []struct {
		CampaignID        string  `json:"campaignId"`
		CampaignName      string  `json:"campaignName"`
		Cost              float64 `json:"cost"`
		Revenue           float64 `json:"revenue"`
		TrafficSourceID   string  `json:"trafficSourceId"`
		TrafficSourceName string  `json:"trafficSourceName"`
	} `json:"rows"`
}

var VoluumApiAccessToken AuthApiResponse

func getAccessToken() string {

	if VoluumApiAccessToken.Token != "" {
		return VoluumApiAccessToken.Token
	}

	authApiRequest := AuthApiRequest{
		AccessID:  config.Configurations.VoluumAccessId,
		AccessKey: config.Configurations.VoluumAccessKey,
	}

	byteArray, err := json.Marshal(authApiRequest)
	if err != nil {
		fmt.Println(err)
	}
	reader := bytes.NewReader(byteArray)
	fmt.Println("Calling Voluum Access Token api")
	req, err := http.NewRequest("POST", "https://api.voluum.com/auth/access/session", reader)
	if err != nil {
		// handle err
	}
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req.Header.Set("Accessid", config.Configurations.VoluumAccessId)
	req.Header.Set("Accesskey", config.Configurations.VoluumAccessKey)
	req.Header.Set("Authorization", "Basic dm9sdXVtZGVtb0B2b2x1dW0uY29tOjFxYXohUUFaIn0=")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// handle err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err.Error())
	}

	err = json.Unmarshal(body, &VoluumApiAccessToken)
	if err != nil {
		fmt.Println("whoops:", err)
	}
	return VoluumApiAccessToken.Token
}

func GetVoluumReportsForMentionedDates(fromDate string, toDate string) (DailyReport, int) {
	token := getAccessToken()
	fmt.Println("Calling Get Vollum Report api for dates from: " + fromDate + " to: " + toDate)
	req, err := http.NewRequest("GET", "https://api.voluum.com/report?include="+configs.Configurations.IncludeTrafficSources+"&limit=10000&groupBy=traffic_source_id&groupBy=campaign_id&from="+fromDate+"&to="+toDate+"&column=traffic_source_id&column=traffic_source&column=campaign_id&column=campaign&column=cost&column=revenue", nil)
	if err != nil {
		// handle err
	}
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req.Header.Set("Cwauth-Token", token)
	req.Header.Set("Authorization", "Basic dm9sdXVtZGVtb0B2b2x1dW0uY29tOjFxYXohUUFaIn0=")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// handle err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err.Error())
	}

	var dailyReport DailyReport
	err = json.Unmarshal(body, &dailyReport)
	if err != nil {
		fmt.Println("whoops:", err)
	}
	return dailyReport, dailyReport.TotalRows
}

func floatToString(inputNum float64) string {
	// to convert a float number to a string
	return strconv.FormatFloat(inputNum, 'f', 6, 64)
}

func createFinalReportForThisMonthData(dailyReport DailyReport, FinalRowsCount int, Day int, month string) ([][]interface{}, int) {
	var values [][]interface{}
	LocalDay := Day

	var firstRowOfSheetLabels []interface{}
	firstRowOfSheetLabels = append(firstRowOfSheetLabels, "Traffic Source Name", "Traffic Source ID", "Campaign Name", "Campaign ID")
	for LocalDay > 1 {
		firstRowOfSheetLabels = append(firstRowOfSheetLabels, "Cost - "+strconv.Itoa(LocalDay-1)+"/"+month, "Revenue - "+strconv.Itoa(LocalDay-1)+"/"+month)
		LocalDay--
	}
	values = append(values, firstRowOfSheetLabels)
	var secondBlankRow []interface{}
	secondBlankRow = append(secondBlankRow, "")
	values = append(values, secondBlankRow)

	ShortlistedTrafficSources = getShortlistedTrafficSources()
	fmt.Println("Preparing final sheet to be pushed to Google Sheets")

	rowID := 0
	for rowID < FinalRowsCount {
		LocalDay = Day
		if ShortlistedTrafficSources[strings.ToLower(dailyReport.Rows[rowID].TrafficSourceName)] || !config.Configurations.TrafficSourceFilteringEnabled {
			var row []interface{}
			row = append(row, dailyReport.Rows[rowID].TrafficSourceName, dailyReport.Rows[rowID].TrafficSourceID, dailyReport.Rows[rowID].CampaignName, dailyReport.Rows[rowID].CampaignID)
			for LocalDay > 1 {
				var cost string
				var revenue string
				if val1, ok := finalMapCost[dailyReport.Rows[rowID].CampaignID+strconv.Itoa(LocalDay)]; ok {
					cost = "$" + val1
				}
				if val2, ok2 := finalMapRevenue[dailyReport.Rows[rowID].CampaignID+strconv.Itoa(LocalDay)]; ok2 {
					revenue = "$" + val2
				}
				row = append(row, cost, revenue)
				LocalDay--
			}
			values = append(values, row)
		}
		rowID++
	}
	return values, rowID
}

var (
	finalMapCost              = make(map[string]string)
	finalMapRevenue           = make(map[string]string)
	ShortlistedTrafficSources = make(map[string]bool)
)

func getShortlistedTrafficSources() map[string]bool {
	configTrafficSources := config.Configurations.TrafficSourcesShortlisted
	for _, source := range configTrafficSources {
		ShortlistedTrafficSources[strings.ToLower(source)] = true
	}
	return ShortlistedTrafficSources
}

func addCostAndRevenueDayWiseToMap(dailyReport DailyReport, Day string) {
	ShortlistedTrafficSources = getShortlistedTrafficSources()
	fmt.Println("Saving Costs and Revenue Day wise")
	rowID := 0
	for rowID < len(dailyReport.Rows) {
		if ShortlistedTrafficSources[strings.ToLower(dailyReport.Rows[rowID].TrafficSourceName)] || !config.Configurations.TrafficSourceFilteringEnabled {
			finalMapCost[dailyReport.Rows[rowID].CampaignID+Day] = floatToString(dailyReport.Rows[rowID].Cost)
			finalMapRevenue[dailyReport.Rows[rowID].CampaignID+Day] = floatToString(dailyReport.Rows[rowID].Revenue)
		}
		rowID++
	}
}

func GetStandardVoluumReport() ([][]interface{}, int, string) {
	var finalValuesToSheet [][]interface{}
	var dailyReport DailyReport
	var RowCount int
	var monthYearDate string
	var EndOfMonthFlag bool
	var currentMonth string

	currentTime := time.Now()
	// currentTime := time.Date(2020, time.August, 1, 18, 59, 59, 0, time.UTC) //This can be used to manually fill a sheet with from desired date
	currentDate := currentTime.Day()
	if currentDate == 1 {
		monthYearDate = currentTime.AddDate(0, -1, 0).Month().String() + strconv.Itoa(currentTime.Year()) //This will be used as Google Sheet name
		EndOfMonthFlag = true
		currentDate = 31
	} else {
		monthYearDate = currentTime.Month().String() + strconv.Itoa(currentTime.Year()) //This will be used as Google Sheet name
	}

	jdayIterator := 0
	for currentDate > 1 {
		fromDate := currentTime.AddDate(0, 0, jdayIterator-1).Format("2006-01-02T00")
		toDate := currentTime.AddDate(0, 0, jdayIterator).Format("2006-01-02T00")
		dailyReport, RowCount = GetVoluumReportsForMentionedDates(fromDate, toDate)
		addCostAndRevenueDayWiseToMap(dailyReport, strconv.Itoa(currentDate))
		currentDate--
		jdayIterator--
	}
	if EndOfMonthFlag {
		fromDate := currentTime.AddDate(0, -1, 0).Format("2006-01-02T00")
		toDate := currentTime.Format("2006-01-02T00")
		dailyReport, RowCount = GetVoluumReportsForMentionedDates(fromDate, toDate)
		currentDate = currentTime.AddDate(0, 0, -1).Day() + 1
		currentMonth = strconv.Itoa(int(currentTime.AddDate(0, -1, 0).Month()))
	} else {
		fromDate := currentTime.AddDate(0, 0, -currentTime.Day()+1).Format("2006-01-02T00")
		toDate := currentTime.Format("2006-01-02T00")
		dailyReport, RowCount = GetVoluumReportsForMentionedDates(fromDate, toDate)
		currentDate = currentTime.Day()
		currentMonth = strconv.Itoa(int(currentTime.Month()))
	}

	finalValuesToSheet, RowCount = createFinalReportForThisMonthData(dailyReport, RowCount, currentDate, currentMonth)
	return finalValuesToSheet, RowCount, monthYearDate
}
