package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const AZURE_HOST = "https://prices.azure.com/api/retail/prices"

type Item struct {
	CurrencyCode         string    `json:"currencyCode"`
	TierMinimumUnits     int       `json:"tierMinimumUnits"`
	RetailPrice          float64   `json:"retailPrice"`
	UnitPrice            float64   `json:"unitPrice"`
	ArmRegionName        string    `json:"armRegionName"`
	Location             string    `json:"location"`
	EffectiveStartDate   time.Time `json:"effectiveStartDate"`
	MeterId              string    `json:"meterId"`
	MeterName            string    `json:"meterName"`
	ProductId            string    `json:"productId"`
	SkuId                string    `json:"skuId"`
	ProductName          string    `json:"productName"`
	SkuName              string    `json:"skuName"`
	ServiceName          string    `json:"serviceName"`
	ServiceId            string    `json:"serviceId"`
	ServiceFamily        string    `json:"serviceFamily"`
	UnitOfMeasure        string    `json:"unitOfMeasure"`
	ItemType             string    `json:"type"`
	IsPrimaryMeterRegion bool      `json:"isPrimaryMeterRegion"`
	ArmSkuName           string    `json:"armSkuName"`
}

type QueryResponse struct {
	BillingCurrency    string `json:"BillingCurrency"`
	CustomerEntityId   string `json:"CustomerEntityId"`
	CustomerEntityType string `json:"CustomerEntityType"`
	Items              []Item `json:"Items"`
}

type Currency uint

const (
	USD Currency = iota
	AUD
	BRL
	CAD
	CHF
	CNY
	DKK
	EUR
	GBP
	INR
	JPY
	KRW
	NOK
	NZD
	RUB
	SEK
	TWD
)

func (c Currency) String() (string, error) {
	switch c {
	case USD:
		return "USD", nil
	case BRL:
		return "BRL", nil
	case CAD:
		return "CAD", nil
	case CHF:
		return "CHF", nil
	case CNY:
		return "CNY", nil
	case DKK:
		return "DKK", nil
	case EUR:
		return "EUR", nil
	case GBP:
		return "GBP", nil
	case INR:
		return "INR", nil
	case JPY:
		return "JPY", nil
	case KRW:
		return "KRW", nil
	case NOK:
		return "NOK", nil
	case NZD:
		return "NZD", nil
	case RUB:
		return "RUB", nil
	case SEK:
		return "SEK", nil
	case TWD:
		return "TWD", nil
	default:
		return "", errors.New(fmt.Sprintf("Invalid currency: %d", c))
	}
}

type QueryFilter struct {
	armRegionName string
	location      string
	meterId       string
	productId     string
	skuId         string
	productName   string
	skuName       string
	serviceName   string
	serviceId     string
	serviceFamily string
	priceType     string
	armSkuName    string
	currencyCode  Currency
}

func (q *QueryFilter) String() (string, error) {
	builder := strings.Builder{}

	currency, err := q.currencyCode.String()
	if err != nil {
		return "", err
	}

	params := url.Values{}

	params.Add("currencyCode", currency)

	if q.armRegionName != "" {
		builder.WriteString(fmt.Sprintf("armRegionName eq '%s' and ", q.armRegionName))
	}
	if q.armSkuName != "" {
		builder.WriteString(fmt.Sprintf("armSkuName eq '%s' and ", q.armSkuName))
	}
	if q.location != "" {
		builder.WriteString(fmt.Sprintf("location eq '%s' and ", q.location))
	}
	if q.meterId != "" {
		builder.WriteString(fmt.Sprintf("meterId eq '%s' and ", q.meterId))
	}
	if q.priceType != "" {
		builder.WriteString(fmt.Sprintf("priceType eq '%s' and ", q.priceType))
	}
	if q.productId != "" {
		builder.WriteString(fmt.Sprintf("productId eq '%s' and ", q.productId))
	}
	if q.productName != "" {
		builder.WriteString(fmt.Sprintf("productName eq '%s' and ", q.productName))
	}
	if q.serviceFamily != "" {
		builder.WriteString(fmt.Sprintf("serviceFamily eq '%s' and ", q.serviceFamily))
	}
	if q.serviceId != "" {
		builder.WriteString(fmt.Sprintf("serviceId eq '%s' and ", q.serviceId))
	}
	if q.serviceName != "" {
		builder.WriteString(fmt.Sprintf("serviceName eq '%s' and ", q.serviceName))
	}
	if q.skuId != "" {
		builder.WriteString(fmt.Sprintf("skuId eq '%s' and ", q.skuId))
	}
	if q.skuName != "" {
		builder.WriteString(fmt.Sprintf("skuName eq '%s'", q.skuName))
	}

	finalFilterString := builder.String()

	if strings.HasSuffix(builder.String(), " and ") {
		finalFilterString = strings.TrimSuffix(builder.String(), " and ")
	}

	if finalFilterString != "" {
		params.Add("$filter", finalFilterString)
	}

	return params.Encode(), nil
}

type CostApi interface {
	Query(q QueryFilter) (QueryResponse, error)
}

type AzureCostApi struct {
}

/*
 * TODO: Return all pages if number of results are > 100. Maybe include an optional "limit" parameter to limit the amount of returned results?
 */
func (a *AzureCostApi) Query(q QueryFilter) (QueryResponse, error) {
	filterString, err := q.String()

	if err != nil {
		return QueryResponse{}, err
	}

	queryUrl := fmt.Sprintf("%s?%s", AZURE_HOST, filterString)
	fmt.Printf("Azure query URL: %s\n\n", queryUrl)

	res, err := http.Get(queryUrl)

	if err != nil {
		return QueryResponse{}, err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return QueryResponse{}, err
	}

	response := QueryResponse{}

	err = json.Unmarshal(body, &response)
	return response, nil
}

func NewApi() CostApi {
	return &AzureCostApi{}
}
