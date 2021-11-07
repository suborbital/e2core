package directive

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rcap"
)

var (
	ErrQueryTypeUnknown = errors.New("unable to determine query type")
	ErrDBTypeUnknown    = errors.New("unable to determine query's database type")
)

const (
	queryTypeInsert = "insert"
	queryTypeSelect = "select"
)

type DBQuery struct {
	Type     string `yaml:"type,omitempty" json:"type,omitempty"`
	Name     string `yaml:"name" json:"name"`
	VarCount int    `yaml:"varCount,omitempty" json:"varCount,omitempty"`
	Query    string `yaml:"query" json:"query"`
}

func (d *DBQuery) toRCAPQuery(dbType string) (*rcap.Query, error) {
	if d.VarCount == 0 {
		count, err := d.varCount(dbType)
		if err != nil {
			return nil, errors.Wrap(err, "failed to varCount")
		}

		d.VarCount = count
	}

	qType, err := d.queryType()
	if err != nil {
		return nil, errors.Wrap(err, "failed to queryType")
	}

	q := &rcap.Query{
		Type:     qType,
		Name:     d.Name,
		VarCount: d.VarCount,
		Query:    d.Query,
	}

	return q, nil
}

func (d *DBQuery) varCount(dbType string) (int, error) {
	var regexString string
	switch dbType {
	case rcap.DBTypeMySQL:
		regexString = ` \? | \?,|\(\?|\?\)|,\?|\? `
	case rcap.DBTypePostgres:
		regexString = `\$\d`
	default:
		return -1, ErrDBTypeUnknown
	}

	rgx, err := regexp.Compile(regexString)
	if err != nil {
		return -1, errors.Wrap(err, "failed to Compile")
	}

	matches := rgx.FindAllString(d.Query, -1)

	return len(matches), nil
}

func (d *DBQuery) queryType() (rcap.QueryType, error) {
	if d.Type != "" {
		switch d.Type {
		case queryTypeInsert:
			return rcap.QueryTypeInsert, nil
		case queryTypeSelect:
			return rcap.QueryTypeSelect, nil
		default:
			return rcap.QueryType(-1), ErrQueryTypeUnknown
		}
	}

	if strings.HasPrefix(d.Query, "insert") || strings.HasPrefix(d.Query, "INSERT") {
		return rcap.QueryTypeInsert, nil
	} else if strings.HasPrefix(d.Query, "select") || strings.HasPrefix(d.Query, "SELECT") {
		return rcap.QueryTypeSelect, nil
	}

	return rcap.QueryType(-1), ErrQueryTypeUnknown
}
