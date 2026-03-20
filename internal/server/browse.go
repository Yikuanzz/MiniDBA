package server

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"mini-dba/internal/sqlrun"
)

const browseFilterValMaxLen = 200

var browsePageSizeChoices = []int{10, 20, 50, 100}

func browseSizeChoices(maxSize int) []int {
	var out []int
	for _, c := range browsePageSizeChoices {
		if c <= maxSize {
			out = append(out, c)
		}
	}
	if len(out) == 0 && maxSize > 0 {
		out = []int{maxSize}
	}
	return out
}

func parseBrowsePS(psParam string, defaultSize, maxSize int) int {
	if maxSize <= 0 {
		maxSize = defaultSize
	}
	if defaultSize > maxSize {
		defaultSize = maxSize
	}
	choices := browseSizeChoices(maxSize)
	if len(choices) == 0 {
		return maxSize
	}
	if v, err := strconv.Atoi(strings.TrimSpace(psParam)); err == nil {
		for _, c := range choices {
			if c == v {
				return c
			}
		}
	}
	d := defaultSize
	for _, c := range choices {
		if c == d {
			return c
		}
	}
	var best int
	found := false
	for _, c := range choices {
		if c <= d {
			if !found || c > best {
				best = c
				found = true
			}
		}
	}
	if found {
		return best
	}
	return choices[0]
}

func escapeLikePatternWithBang(val string) string {
	val = strings.ReplaceAll(val, "!", "!!")
	val = strings.ReplaceAll(val, "%", "!%")
	val = strings.ReplaceAll(val, "_", "!_")
	return val
}

// buildBrowseFilter 返回 SQL 片段 " WHERE ..." 与参数（无筛选时片段为空）。
func buildBrowseFilter(fcol, fop, fval string, colSet map[string]struct{}) (whereSQL string, args []interface{}, err error) {
	fcol = strings.TrimSpace(fcol)
	if fcol == "" {
		return "", nil, nil
	}
	fval = strings.TrimSpace(fval)
	fop = strings.TrimSpace(fop)
	if fop == "" {
		return "", nil, fmt.Errorf("筛选须指定列与规则")
	}
	if err := sqlrun.ValidateIdent(fcol); err != nil {
		return "", nil, err
	}
	if _, ok := colSet[fcol]; !ok {
		return "", nil, fmt.Errorf("未知列")
	}
	if len(fval) > browseFilterValMaxLen {
		return "", nil, fmt.Errorf("筛选值过长")
	}
	col := "`" + fcol + "`"
	switch strings.ToLower(fop) {
	case "contains":
		pat := "%" + escapeLikePatternWithBang(fval) + "%"
		return fmt.Sprintf(" WHERE %s LIKE ? ESCAPE '!'", col), []interface{}{pat}, nil
	case "prefix":
		pat := escapeLikePatternWithBang(fval) + "%"
		return fmt.Sprintf(" WHERE %s LIKE ? ESCAPE '!'", col), []interface{}{pat}, nil
	case "suffix":
		pat := "%" + escapeLikePatternWithBang(fval)
		return fmt.Sprintf(" WHERE %s LIKE ? ESCAPE '!'", col), []interface{}{pat}, nil
	case "eq":
		return fmt.Sprintf(" WHERE %s = ?", col), []interface{}{fval}, nil
	default:
		return "", nil, fmt.Errorf("未知筛选规则")
	}
}

func browseTableColumns(ctx context.Context, db *sql.DB, table string) ([]string, map[string]struct{}, error) {
	q := fmt.Sprintf("DESCRIBE `%s`", table)
	dres, _, err := sqlrun.Run(ctx, db, q, true, 2000)
	if err != nil {
		return nil, nil, err
	}
	colSet := make(map[string]struct{})
	var names []string
	for _, row := range dres.Rows {
		if len(row) == 0 {
			continue
		}
		name := row[0]
		if err := sqlrun.ValidateIdent(name); err != nil {
			continue
		}
		colSet[name] = struct{}{}
		names = append(names, name)
	}
	if len(names) == 0 {
		return nil, nil, fmt.Errorf("无法读取列信息")
	}
	return names, colSet, nil
}

func parseSortDir(d string) string {
	if strings.EqualFold(d, "desc") {
		return "DESC"
	}
	return "ASC"
}

func cloneURLValues(v url.Values) url.Values {
	out := make(url.Values)
	for k, vs := range v {
		out[k] = append([]string(nil), vs...)
	}
	return out
}

func (s *Server) buildBrowseURL(v url.Values) string {
	return s.absPath("/browse?" + v.Encode())
}
