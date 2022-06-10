package directive

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/suborbital/velocity/capabilities"
)

var (
	ErrQueryTypeUnknown = errors.New("unable to determine query type")
	ErrDBTypeUnknown    = errors.New("unable to determine query's database type")
)

const (
	queryTypeInsert = "insert"
	queryTypeSelect = "select"
	queryTypeUpdate = "update"
	queryTypeDelete = "delete"
)

type DBQuery struct {
	Type     string `yaml:"type,omitempty" json:"type,omitempty"`
	Name     string `yaml:"name" json:"name"`
	VarCount int    `yaml:"varCount,omitempty" json:"varCount,omitempty"`
	Query    string `yaml:"query" json:"query"`
}

func (d *DBQuery) toRCAPQuery(dbType string) (*capabilities.Query, error) {
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

	q := &capabilities.Query{
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
	case capabilities.DBTypeMySQL:
		regexString = ` \? | \?,|\(\?|\?\)|,\?|\? `
	case capabilities.DBTypePostgres:
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

func (d *DBQuery) queryType() (capabilities.QueryType, error) {
	if d.Type != "" {
		switch d.Type {
		case queryTypeInsert:
			return capabilities.QueryTypeInsert, nil
		case queryTypeSelect:
			return capabilities.QueryTypeSelect, nil
		case queryTypeUpdate:
			return capabilities.QueryTypeUpdate, nil
		case queryTypeDelete:
			return capabilities.QueryTypeDelete, nil
		default:
			return capabilities.QueryType(-1), ErrQueryTypeUnknown
		}
	}

	if strings.HasPrefix(d.Query, "insert") || strings.HasPrefix(d.Query, "INSERT") {
		return capabilities.QueryTypeInsert, nil
	} else if strings.HasPrefix(d.Query, "select") || strings.HasPrefix(d.Query, "SELECT") {
		return capabilities.QueryTypeSelect, nil
	} else if strings.HasPrefix(d.Query, "update") || strings.HasPrefix(d.Query, "UPDATE") {
		return capabilities.QueryTypeUpdate, nil
	} else if strings.HasPrefix(d.Query, "delete") || strings.HasPrefix(d.Query, "DELETE") {
		return capabilities.QueryTypeDelete, nil
	}

	return capabilities.QueryType(-1), ErrQueryTypeUnknown
}
