package cmd

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/davecgh/go-spew/spew"
	"github.com/spf13/cobra"
)

var sgufiltercmd = &cobra.Command{
	Use:   "filter",
	Short: "Filters htm against utilization factor of column args are: filter name and filter values. Allowed filters are: lt, gt, uniq",
	Long:  "Allowed columns are: profile, material, caseuy, caseuz, case, uz, uy, lay, laz, utilization, bar",
	Run:   slsFilterCmdRun,
	Args:  cobra.MinimumNArgs(2),
}

var (
	OutputFileName string
	InputFileName  string
	OutputColumn   string
	ColumnName     string
	Verbose        *bool
)

func init() {
	// sgufiltercmd.Flags().StringVarP(&OutputFileName, "out", "o", "out.txt", "Sets output filename in which no of bars will be written")
	sgufiltercmd.Flags().StringVarP(&InputFileName, "in", "i", "robotPrn.htm", "Sets input filename from which data will be read")
	sgufiltercmd.Flags().StringVarP(&ColumnName, "col", "c", "uz", "Sets column for filter, proper colnames are uy, uz, lay, laz, utilization, case")
	sgufiltercmd.Flags().StringVarP(&OutputColumn, "ocol", "y", "bar", "Sets output column, default is bar")
	Verbose = sgufiltercmd.Flags().BoolP("verbose", "v", false, "RobotResult filter is more verbose")
	rootCmd.AddCommand(sgufiltercmd)
}

func slsFilterCmdRun(cmd *cobra.Command, args []string) {
	filterName := args[0]
	filterValue := args[1]
	rows, err := parseHtmlResults(InputFileName)
	if err != nil {
		log.Panicf("Cannot parse results file: %v", err)
		return
	}
	log.Printf("total read number of rows: %d", len(rows))
	if len(rows) < 1 {
		return
	}
	filter := getFilter(filterName, filterValue, ColumnName)
	if filter == nil {
		log.Println("Cannot get proper filter")
		return
	}
	results := make([]*row, 0)
	for _, r := range rows {
		if filter(r) {
			results = append(results, r)
		}
	}
	log.Println("Number of filtered results:", len(results))
	printResults(results)
}

func printResults(results []*row) {
	if *Verbose {
		spew.Dump(results)
	}
	rp := make([]string, len(results))
	for i, r := range results {
		val, err := r.ValByCol(OutputColumn)
		if err != nil {
			log.Printf("Cannot retreive value for output col: %v", OutputColumn)
			return
		}
		rp[i] = fmt.Sprintf("%v", val)
	}
	res := strings.Join(rp, " ")
	fmt.Println(res)
}

type filter func(*row) bool

func getFilter(filterName string, filterValue string, colName string) filter {
	var r row
	if filterName == "lt" {
		if _, err := r.FloatValByColName(colName); err != nil {
			log.Printf("Nie mogę zastosować filtru 'lt' dla kolumny %s", colName)
			return nil
		}
		return getLowerThanFilter(filterValue, colName)
	} else if filterName == "gt" {
		if _, err := r.FloatValByColName(colName); err != nil {
			log.Printf("Nie mogę zastosować filtru 'lt' dla kolumny %s", colName)
			return nil
		}
		return getGreaterThanFilter(filterValue, colName)
	} else if filterName == "uniq" {
		if _, err := r.StringValByColName(colName); err != nil {
			log.Printf("Nie mogę zastosować filtru 'uniq' dla kolumnt %s", colName)
		}
		return getUniqFilter(colName)
	}
	return nil
}

func getUniqFilter(cname string) filter {
	s := make(map[interface{}]struct{})
	return func(val *row) bool {
		colval, err := val.ValByCol(cname)
		if err != nil {
			return false
		}
		_, ok := s[colval]
		if !ok {
			s[colval] = struct{}{}
			return true
		}
		return false
	}
}

func getGreaterThanFilter(filterValString string, colName string) filter {
	filterVal, err := strconv.ParseFloat(filterValString, 64)
	if err != nil {
		return nil
	}
	return func(val *row) bool {
		v, err := val.FloatValByColName(colName)
		if err != nil {
			return false
		}
		return v > filterVal
	}
}

func getLowerThanFilter(filterValString string, colName string) filter {
	filterVal, err := strconv.ParseFloat(filterValString, 64)
	if err != nil {
		return nil
	}
	return func(val *row) bool {
		v, err := val.FloatValByColName(colName)
		if err != nil {
			return false
		}
		return v < filterVal
	}
}

func parseHtmlResults(filename string) ([]*row, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	doc, err := goquery.NewDocumentFromReader(file)
	if err != nil {
		return nil, err
	}

	rows := make([]*row, 0)
	var colMap map[int]string
	doc.Find("tr").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			colMap = mapColumns(s)
		} else {
			if r, err := NewRowFromSelection(s, colMap); err == nil {
				rows = append(rows, r)
			}
		}

	})
	return rows, nil
}

func mapColumns(headers *goquery.Selection) map[int]string {
	mp := make(map[int]string)
	headers.Find("td").Each(func(i int, td *goquery.Selection) {
		colName := strings.TrimSpace(td.Text())
		if len(colName) > 0 {
			mp[i] = strings.TrimSpace(td.Text())
		}
	})
	return mp
}

type row struct {
	Bar      int64
	Profile  string
	Material string
	// To jest do SLS
	Uy     float64
	CaseUy string
	Uz     float64
	CaseUz string
	// To jest do ULS
	Lay         float64
	Laz         float64
	Utilization float64
	Case        string
}

func (r *row) ValByCol(colName string) (interface{}, error) {
	if val, err := r.StringValByColName(colName); err == nil {
		return val, nil
	}
	if val, err := r.FloatValByColName(colName); err == nil {
		return val, nil
	}
	if val, err := r.IntValByColName(colName); err == nil {
		return val, nil
	}
	return nil, fmt.Errorf("no such column: %v", colName)
}

func (r *row) IntValByColName(colName string) (int64, error) {
	cname := strings.TrimSpace(strings.ToLower(colName))
	if cname == "bar" {
		return r.Bar, nil
	}
	return -1, fmt.Errorf("no such int column: %v", colName)
}

func (r *row) StringValByColName(colName string) (string, error) {
	cname := strings.TrimSpace(strings.ToLower(colName))
	if cname == "profile" {
		return r.Profile, nil
	} else if cname == "material" {
		return r.Material, nil
	} else if cname == "caseuy" {
		return r.CaseUy, nil
	} else if cname == "caseuz" {
		return r.CaseUz, nil
	} else if cname == "case" {
		return r.Case, nil
	}
	return "", fmt.Errorf("no such string column: %v", colName)
}

func (r *row) FloatValByColName(colName string) (float64, error) {
	cname := strings.TrimSpace(strings.ToLower(colName))
	if cname == "uy" {
		return r.Uy, nil
	} else if cname == "uz" {
		return r.Uz, nil
	} else if cname == "lay" {
		return r.Lay, nil
	} else if cname == "laz" {
		return r.Laz, nil
	} else if cname == "utilization" {
		return r.Utilization, nil
	}
	return -1, fmt.Errorf("no such float64 column: %v", colName)
}

func (r *row) ApplyValue(column string, value interface{}) error {
	vals := value.(string)
	vals = strings.TrimSpace(vals)
	var err error
	if column == "Pręt" {
		if err := r.ApplyBarValue(vals); err != nil {
			return err
		}
	} else if column == "Profil" {
		r.Profile = vals
	} else if column == "Materiał" {
		r.Material = vals
	} else if column == "Prop.(uy)" {
		r.Uy, err = strconv.ParseFloat(vals, 64)
		if err != nil {
			return err
		}
	} else if column == "Przyp.(uy)" {
		if err := r.ApplyCaseUyValue(vals); err != nil {
			return err
		}
	} else if column == "Prop.(uz)" {
		r.Uz, err = strconv.ParseFloat(vals, 64)
		if err != nil {
			return err
		}
	} else if column == "Przyp.(uz)" {
		if err := r.ApplyCaseUzValue(vals); err != nil {
			return err
		}
	} else if column == "Lay" {
		r.Lay, err = strconv.ParseFloat(vals, 64)
		if err != nil {
			return err
		}
	} else if column == "Laz" {
		r.Laz, err = strconv.ParseFloat(vals, 64)
		if err != nil {
			return err
		}
	} else if column == "Wytęż." {
		r.Utilization, err = strconv.ParseFloat(vals, 64)
		if err != nil {
			return err
		}
	} else if column == "Przypadek" {
		if err := r.ApplyCaseValue(vals); err != nil {
			return err
		}
	}

	return nil
}

func (r *row) ApplyBarValue(val string) error {
	barNo, err := getFirstInt(val)
	if err != nil {
		return err
	}
	r.Bar = barNo
	return nil
}

func (r *row) ApplyCaseValue(val string) error {
	caseNo, err := getFirstInt(val)
	if err != nil {
		return err
	}
	r.Case = fmt.Sprintf("%d", caseNo)
	return nil
}

func (r *row) ApplyCaseUyValue(val string) error {
	caseNo, err := getFirstInt(val)
	if err != nil {
		return err
	}
	r.CaseUy = fmt.Sprintf("%d", caseNo)
	return nil
}

func (r *row) ApplyCaseUzValue(val string) error {
	caseNo, err := getFirstInt(val)
	if err != nil {
		return err
	}
	r.CaseUz = fmt.Sprintf("%d", caseNo)
	return nil
}

func getFirstInt(val string) (int64, error) {
	var txt = strings.TrimSpace(val)
	match := re.FindString(txt)
	match = strings.TrimSpace(match)
	return strconv.ParseInt(match, 10, 64)
}

var re = regexp.MustCompile(`(?m)^(?P<barno>\d+)\s`)

func NewRowFromSelection(s *goquery.Selection, colMap map[int]string) (*row, error) {
	var r row
	var err error
	s.Find("td").Each(func(i int, sm *goquery.Selection) {
		err = r.ApplyValue(colMap[i], sm.Text())
	})
	if err != nil {
		return nil, err
	} else {
		return &r, nil
	}
}
