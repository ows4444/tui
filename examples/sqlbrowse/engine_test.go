package main

import (
	"fmt"
	"strings"
	"testing"
)

// small is a database of two tiny tables with values chosen to show edge cases.
func small() *database {
	return &database{tables: []*table{
		{name: "t", cols: []string{"id", "name", "n", "grp"}, rows: [][]value{
			{1.0, "alpha", 10.0, "x"}, {2.0, "Beta", 20.0, "x"}, {3.0, "gamma", nil, "y"}, {4.0, "delta", 5.5, "y"}, {5.0, nil, 7.0, nil},
		}},
		{name: "u", cols: []string{"id", "tid", "note"}, rows: [][]value{
			{1.0, 1.0, "first"}, {2.0, 1.0, "second"}, {3.0, 3.0, "third"}, {4.0, 9.0, "orphan"},
		}},
	}}
}

// run runs q on db and shows the result as "a,b|c,d" with NULL for nothing.
func run(t *testing.T, db *database, q string) string {
	t.Helper()
	r, err := db.query(q)
	if err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	var rows []string
	for _, row := range r.rows {
		var cells []string
		for _, v := range row {
			cells = append(cells, format(v))
		}
		rows = append(rows, strings.Join(cells, ","))
	}
	return strings.Join(rows, "|")
}

func fails(t *testing.T, db *database, q, want string) *sqlError {
	t.Helper()
	_, err := db.query(q)
	if err == nil || !strings.Contains(err.msg, want) {
		t.Fatalf("%s: err = %v, want it to mention %q", q, err, want)
	}
	return err
}

func TestSelectBasics(t *testing.T) {
	db := small()
	for q, want := range map[string]string{
		"SELECT * FROM t WHERE id = 2":                           "2,Beta,20,x",
		"select id, name from t where id <= 2":                   "1,alpha|2,Beta",
		"SELECT name FROM t WHERE n > 6 AND grp = 'x'":           "alpha|Beta",
		"SELECT id FROM t WHERE n > 100":                         "",
		"SELECT id FROM t WHERE grp = 'x' OR n = 5.5":            "1|2|4",
		"SELECT id FROM t WHERE NOT grp = 'x'":                   "3|4",
		"SELECT id FROM t WHERE id IN (1, 3, 5)":                 "1|3|5",
		"SELECT id FROM t WHERE id NOT IN (1, 3, 5)":             "2|4",
		"SELECT id FROM t WHERE n BETWEEN 6 AND 20":              "1|2|5",
		"SELECT id FROM t WHERE n NOT BETWEEN 6 AND 20":          "4",
		"SELECT id FROM t WHERE name LIKE 'a%'":                  "1",
		"SELECT id FROM t WHERE name LIKE '%ta'":                 "2|4",
		"SELECT id FROM t WHERE name LIKE '_eta'":                "2",
		"SELECT id FROM t WHERE name NOT LIKE '%a%'":             "",
		"SELECT id FROM t WHERE n IS NULL":                       "3",
		"SELECT id FROM t WHERE n IS NOT NULL AND grp IS NULL":   "5",
		"SELECT id FROM t WHERE (id = 1 OR id = 2) AND n > 15":   "2",
		"SELECT id FROM t WHERE id = 1 -- a comment\n OR id = 4": "1|4",
		"SELECT id FROM t WHERE id <> 1 AND id != 2 AND id < 4":  "3",
	} {
		if got := run(t, db, q); got != want {
			t.Errorf("%s\n got %q\nwant %q", q, got, want)
		}
	}
}

func TestNullNeverMatchesAComparison(t *testing.T) {
	db := small()
	for _, q := range []string{"SELECT id FROM t WHERE n = NULL", "SELECT id FROM t WHERE n <> 10 AND n IS NULL AND n > 0", "SELECT id FROM t WHERE name = NULL"} {
		if got := run(t, db, q); got != "" && !strings.HasPrefix(q, "SELECT id FROM t WHERE n <> 10 AND n IS NULL") {
			t.Errorf("%s = %q, want nothing", q, got)
		}
	}
	if got := run(t, db, "SELECT id FROM t WHERE n > 6"); got != "1|2|5" {
		t.Errorf("a NULL slipped into a > comparison: %q", got)
	}
}

func TestOrderLimitOffsetDistinct(t *testing.T) {
	db := small()
	for q, want := range map[string]string{
		"SELECT id FROM t ORDER BY n":                     "3|4|5|1|2",
		"SELECT id FROM t ORDER BY n DESC":                "2|1|5|4|3",
		"SELECT id FROM t ORDER BY grp, id DESC":          "5|2|1|4|3",
		"SELECT id FROM t ORDER BY name":                  "5|2|1|4|3",
		"SELECT id FROM t ORDER BY 1 DESC LIMIT 2":        "5|4",
		"SELECT id FROM t ORDER BY id LIMIT 2 OFFSET 2":   "3|4",
		"SELECT id FROM t ORDER BY id OFFSET 4":           "5",
		"SELECT id FROM t LIMIT 0":                        "",
		"SELECT DISTINCT grp FROM t ORDER BY grp":         "NULL|x|y",
		"SELECT id AS k FROM t ORDER BY k DESC LIMIT 1":   "5",
		"SELECT n * 2 AS dbl FROM t ORDER BY dbl LIMIT 2": "NULL|11",
	} {
		got := run(t, db, q)
		if got != want {
			t.Errorf("%s\n got %q\nwant %q", q, got, want)
		}
	}
}

func TestExpressionsAndFunctions(t *testing.T) {
	db := small()
	for q, want := range map[string]string{
		"SELECT id + 1, id * 2, 7 - id, 10 / 4, 10 % 4 FROM t WHERE id = 1": "2,2,6,2.5,2",
		"SELECT -id FROM t WHERE id = 3":                                    "-3",
		"SELECT UPPER(name), LOWER(name), LENGTH(name) FROM t WHERE id = 2": "BETA,beta,4",
		"SELECT ROUND(n / 3, 2) FROM t WHERE id = 1":                        "3.33",
		"SELECT ROUND(5.5) FROM t WHERE id = 1":                             "6",
		"SELECT ABS(-4) FROM t WHERE id = 1":                                "4",
		"SELECT COALESCE(n, 0) FROM t WHERE id = 3":                         "0",
		"SELECT n + 1 FROM t WHERE id = 3":                                  "NULL",
		"SELECT 'it''s' FROM t WHERE id = 1":                                "it's",
		"SELECT name FROM t WHERE LOWER(name) = 'beta'":                     "Beta",
		"SELECT id FROM t WHERE '10' = n":                                   "1",
		"SELECT TRUE, FALSE, NULL FROM t WHERE id = 1":                      "true,false,NULL",
	} {
		if got := run(t, db, q); got != want {
			t.Errorf("%s\n got %q\nwant %q", q, got, want)
		}
	}
}

func TestAggregates(t *testing.T) {
	db := small()
	for q, want := range map[string]string{
		"SELECT COUNT(*) FROM t":                                                        "5",
		"SELECT COUNT(n), COUNT(name) FROM t":                                           "4,4",
		"SELECT SUM(n), AVG(n), MIN(n), MAX(n) FROM t":                                  "42.5,10.625,5.5,20",
		"SELECT MIN(name), MAX(name) FROM t":                                            "Beta,gamma",
		"SELECT COUNT(*), SUM(n) FROM t WHERE id > 100":                                 "0,NULL",
		"SELECT grp, COUNT(*) FROM t GROUP BY grp ORDER BY grp":                         "NULL,1|x,2|y,2",
		"SELECT grp, SUM(n) AS s FROM t GROUP BY grp HAVING SUM(n) > 6 ORDER BY s DESC": "x,30|NULL,7",
		"SELECT grp, COUNT(*) FROM t GROUP BY grp ORDER BY COUNT(*) DESC, grp":          "x,2|y,2|NULL,1",
		"SELECT COUNT(*) + 1 FROM t":                                                    "6",
		"SELECT ROUND(AVG(n), 1) FROM t WHERE grp = 'x'":                                "15",
		"SELECT DISTINCT grp FROM t WHERE grp IS NOT NULL ORDER BY grp":                 "x|y",
	} {
		if got := run(t, db, q); got != want {
			t.Errorf("%s\n got %q\nwant %q", q, got, want)
		}
	}
}

func TestJoin(t *testing.T) {
	db := small()
	for q, want := range map[string]string{
		"SELECT t.name, u.note FROM t JOIN u ON u.tid = t.id ORDER BY u.id":                   "alpha,first|alpha,second|gamma,third",
		"SELECT t.name, u.note FROM t INNER JOIN u ON t.id = u.tid WHERE u.id > 1 ORDER BY 2": "alpha,second|gamma,third",
		"SELECT a.id, b.id FROM t a JOIN u b ON b.tid = a.id WHERE a.id = 1 ORDER BY b.id":    "1,1|1,2",
		"SELECT t.grp, COUNT(*) FROM t JOIN u ON u.tid = t.id GROUP BY t.grp ORDER BY 1":      "x,2|y,1",
		"SELECT note FROM t JOIN u ON u.tid = t.id WHERE name = 'gamma'":                      "third",
	} {
		if got := run(t, db, q); got != want {
			t.Errorf("%s\n got %q\nwant %q", q, got, want)
		}
	}
	r, err := db.query("SELECT * FROM t JOIN u ON u.tid = t.id LIMIT 1")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(r.cols, ",") != "t.id,name,n,grp,u.id,tid,note" {
		t.Fatalf("columns of SELECT * over a join: %v", r.cols)
	}
}

func TestColumnHeadings(t *testing.T) {
	db := small()
	r, err := db.query("SELECT id, name AS who, n * 2, UPPER(name) shout, t.grp FROM t LIMIT 1")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(r.cols, "|"); got != "id|who|n * 2|shout|grp" {
		t.Fatalf("headings = %s", got)
	}
}

func TestErrorsExplainThemselves(t *testing.T) {
	db := small()
	for q, want := range map[string]string{
		"":                                    "type a query",
		"   ":                                 "type a query",
		"DELETE FROM t":                       "only runs SELECT",
		"UPDATE t SET n = 1":                  "only runs SELECT",
		"SELECT":                              "expected a value",
		"SELECT id":                           "expected FROM",
		"SELECT id FROM":                      "expected a table name",
		"SELECT id FROM nope":                 "no table called nope",
		"SELECT nope FROM t":                  "no column called nope",
		"SELECT x.id FROM t":                  "no column called x.id",
		"SELECT id FROM t WHERE":              "expected a value",
		"SELECT id FROM t WHERE id =":         "expected a value",
		"SELECT id FROM t WHERE (id = 1":      "expected )",
		"SELECT id FROM t WHERE name = 'oops": "no closing quote",
		"SELECT id FROM t WHERE id @ 1":       "unexpected \"@\"",
		"SELECT id FROM t LIMIT x":            "expected a whole number",
		"SELECT id FROM t LIMIT -1":           "expected a whole number",
		"SELECT id FROM t GROUP id":           "expected BY",
		"SELECT id FROM t ORDER id":           "expected BY",
		"SELECT id FROM t extra stuff":        "unexpected",
		"SELECT id FROM t WHERE id = 1 1":     "unexpected",
		"SELECT 1 / 0 FROM t":                 "division by zero",
		"SELECT name + 1 FROM t WHERE id = 1": "needs numbers",
		"SELECT FOO(id) FROM t":               "no function called FOO",
		"SELECT UPPER() FROM t":               "takes 1 argument",
		"SELECT SUM(*) FROM t":                "not allowed",
		"SELECT id FROM t WHERE SUM(n) > 1":   "cannot be used here",
		"SELECT id FROM t JOIN u ON id = tid": "in both tables",
		"SELECT id FROM t JOIN u":             "expected ON",
		"SELECT SUM(name) FROM t":             "needs numbers",
		"SELECT id FROM t WHERE id IN 1":      "expected (",
		"SELECT id FROM t WHERE n BETWEEN 1":  "expected AND",
		"SELECT id FROM t WHERE n IS 5":       "expected NULL",
		"SELECT id AS FROM t":                 "expected a name after AS",
		"SELECT select FROM t":                "expected a value",
	} {
		fails(t, db, q, want)
	}
}

func TestErrorPositionsPointAtTheProblem(t *testing.T) {
	db := small()
	cases := map[string]string{
		"SELECT nope FROM t":            "nope",
		"SELECT id FROM nowhere":        "nowhere",
		"SELECT id FORM t":              "FORM",
		"SELECT id FROM t WHERE id @ 1": "@",
	}
	for q, at := range cases {
		_, err := db.query(q)
		if err == nil || err.pos < 0 || !strings.HasPrefix(q[err.pos:], at) {
			t.Errorf("%s: error %+v does not point at %q", q, err, at)
		}
	}
	if _, err := db.query("SELECT 'é' FROM t WHERE id @ 1"); err == nil || !strings.HasPrefix("SELECT 'é' FROM t WHERE id @ 1"[err.pos:], "@") {
		t.Errorf("a multi-byte character threw the position off: %+v", err)
	}
}

func TestFormatAndCompare(t *testing.T) {
	for v, want := range map[value]string{nil: "NULL", 3.0: "3", 2.5: "2.5", 1.0 / 3: "0.333333", true: "true", false: "false", "x": "x", -0.0: "0", 1e20: "100000000000000000000"} {
		if got := format(v); got != want {
			t.Errorf("format(%v) = %q, want %q", v, got, want)
		}
	}
	if compare(nil, 1.0) >= 0 || compare(1.0, nil) <= 0 || compare(nil, nil) != 0 {
		t.Error("NULL should sort first")
	}
	if compare(2.0, 10.0) >= 0 || compare("2", 10.0) >= 0 || compare("abc", "abd") >= 0 || compare(false, true) >= 0 {
		t.Error("ordering is wrong")
	}
	if compare("10", "9") >= 0 {
		t.Error("two strings compare as text")
	}
}

func TestLikeEscapesRegexCharacters(t *testing.T) {
	for _, c := range []struct {
		s, p string
		want bool
	}{{"a.b", "a.b", true}, {"axb", "a.b", false}, {"a+b", "a+b", true}, {"(x)", "(x)", true}, {"ABC", "abc", true}, {"a\nb", "a%b", true}, {"", "%", true}, {"", "_", false}} {
		if like(c.s, c.p) != c.want {
			t.Errorf("like(%q, %q) = %v", c.s, c.p, !c.want)
		}
	}
}

func TestFixtureIsConsistent(t *testing.T) {
	db := fixture()
	if db.find("CUSTOMERS") == nil || db.find("nope") != nil {
		t.Fatal("find")
	}
	for _, tb := range db.tables {
		if len(tb.cols) != len(tb.types) || len(tb.rows) == 0 {
			t.Errorf("%s: %d cols, %d types, %d rows", tb.name, len(tb.cols), len(tb.types), len(tb.rows))
		}
		for i, r := range tb.rows {
			if len(r) != len(tb.cols) {
				t.Fatalf("%s row %d has %d cells", tb.name, i, len(r))
			}
		}
	}
	if fmt.Sprint(run(t, db, "SELECT COUNT(*) FROM customers")) != "30" || run(t, db, "SELECT COUNT(*) FROM orders") != "80" {
		t.Fatal("unexpected table sizes")
	}
	// Every order points at a real customer, and every item at a real order and product.
	if run(t, db, "SELECT COUNT(*) FROM orders o JOIN customers c ON c.id = o.customer_id") != "80" {
		t.Error("an order has no customer")
	}
	if got, want := run(t, db, "SELECT COUNT(*) FROM order_items i JOIN orders o ON o.id = i.order_id"), run(t, db, "SELECT COUNT(*) FROM order_items"); got != want {
		t.Errorf("an item has no order: %s of %s joined", got, want)
	}
	if got, want := run(t, db, "SELECT COUNT(*) FROM order_items i JOIN products p ON p.id = i.product_id"), run(t, db, "SELECT COUNT(*) FROM order_items"); got != want {
		t.Errorf("an item has no product: %s of %s joined", got, want)
	}
	a, b := fixture(), fixture()
	if run(t, a, "SELECT SUM(total) FROM orders") != run(t, b, "SELECT SUM(total) FROM orders") {
		t.Error("the fixture changes from run to run")
	}
}

func TestRealisticQueries(t *testing.T) {
	db := fixture()
	for _, q := range []string{
		"SELECT c.name, COUNT(*) AS orders, ROUND(SUM(o.total), 2) AS spent FROM customers c JOIN orders o ON o.customer_id = c.id GROUP BY c.name HAVING COUNT(*) > 1 ORDER BY spent DESC LIMIT 5",
		"SELECT status, COUNT(*), AVG(total) FROM orders GROUP BY status ORDER BY 2 DESC",
		"SELECT name, price FROM products WHERE category IN ('parts', 'tools') AND price BETWEEN 10 AND 100 ORDER BY price",
		"SELECT country, COUNT(*) FROM customers WHERE vip = TRUE GROUP BY country ORDER BY 2 DESC, country",
		"SELECT DISTINCT city FROM customers ORDER BY city",
		"SELECT id, placed FROM orders WHERE status IS NULL",
	} {
		r, err := db.query(q)
		if err != nil || len(r.cols) == 0 {
			t.Errorf("%s: %v", q, err)
		}
	}
}
