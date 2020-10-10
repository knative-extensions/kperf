package utils

import (
	"encoding/csv"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
)

func GenerateCSVFile(path string, rows [][]string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create csv file %s\n", err)
	}
	defer file.Close()

	csvWriter := csv.NewWriter(file)
	csvWriter.WriteAll(rows)
	csvWriter.Flush()
	return nil
}

func GenerateHTMLFile(sourceCSV string, targetHTML string) error {
	data, err := ioutil.ReadFile(sourceCSV)
	if err != nil {
		return fmt.Errorf("failed to read csv file %s", err)
	}
	htmlTemplate, err := Asset("templates/single_chart.html")
	if err != nil {
		return fmt.Errorf("failed to load asset: %s", err)
	}
	viewTemplate, err := template.New("chart").Parse(string(htmlTemplate))
	if err != nil {
		return fmt.Errorf("failed to parse html template %s", err)
	}
	htmlFile, err := os.OpenFile(targetHTML, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open html file %s", err)
	}
	defer htmlFile.Close()
	return viewTemplate.Execute(htmlFile, map[string]interface{}{
		"Data": string(data),
	})
}
