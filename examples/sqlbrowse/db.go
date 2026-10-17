package main

import "fmt"

// lcg is a tiny repeatable random number generator: the same fixture on every
// run and every machine, with no seed to forget.
type lcg uint32

func (l *lcg) next(n int) int {
	*l = *l*1664525 + 1013904223
	return int(uint32(*l)>>8) % n
}

// fixture is the database the browser opens: a small shop. The tables are
// built in code, so nothing is read from or written to disk.
func fixture() *database {
	r := lcg(2026)
	names := []string{"Ada Lovelace", "Alan Turing", "Grace Hopper", "Edsger Dijkstra", "Barbara Liskov", "Donald Knuth", "Margaret Hamilton",
		"Dennis Ritchie", "Ken Thompson", "Linus Torvalds", "Radia Perlman", "Tim Berners-Lee", "Hedy Lamarr", "John von Neumann", "Katherine Johnson",
		"Claude Shannon", "Annie Easley", "Vint Cerf", "Frances Allen", "Niklaus Wirth", "Sophie Wilson", "Guido van Rossum", "Anita Borg",
		"Bjarne Stroustrup", "Joan Clarke", "Brian Kernighan", "Lynn Conway", "Rob Pike", "Shafi Goldwasser", "James Gosling"}
	places := [][2]string{{"Berlin", "Germany"}, {"Cairo", "Egypt"}, {"Oslo", "Norway"}, {"Tokyo", "Japan"}, {"Lima", "Peru"}, {"Toronto", "Canada"}, {"Mumbai", "India"}, {"Sydney", "Australia"}}
	customers := &table{name: "customers", cols: []string{"id", "name", "city", "country", "signup", "vip"}, types: []string{"int", "text", "text", "text", "date", "bool"}}
	for i, n := range names {
		p := places[r.next(len(places))]
		customers.rows = append(customers.rows, []value{float64(i + 1), n, p[0], p[1], fmt.Sprintf("202%d-%02d-%02d", 3+r.next(3), 1+r.next(12), 1+r.next(28)), r.next(4) == 0})
	}
	products := &table{name: "products", cols: []string{"id", "name", "category", "price", "stock"}, types: []string{"int", "text", "text", "real", "int"}}
	for i, p := range []struct {
		n, c string
		pr   float64
	}{{"Brass gear", "parts", 12.5}, {"Punch cards (100)", "supplies", 15}, {"Steam valve", "parts", 48}, {"Copper wire 10m", "supplies", 9.9}, {"Difference engine", "machines", 4200},
		{"Analytical engine", "machines", 9800}, {"Oil can", "tools", 7.25}, {"Torque wrench", "tools", 39}, {"Lens set", "parts", 120}, {"Ledger book", "supplies", 18},
		{"Brass rivets (1kg)", "parts", 22}, {"Clockwork motor", "machines", 310}, {"Safety goggles", "tools", 14.5}, {"Pressure gauge", "parts", 65}, {"Workshop apron", "supplies", 29}} {
		products.rows = append(products.rows, []value{float64(i + 1), p.n, p.c, p.pr, float64(r.next(200))})
	}
	orders := &table{name: "orders", cols: []string{"id", "customer_id", "total", "status", "placed"}, types: []string{"int", "int", "real", "text", "date"}}
	items := &table{name: "order_items", cols: []string{"order_id", "product_id", "qty"}, types: []string{"int", "int", "int"}}
	statuses := []string{"paid", "paid", "paid", "shipped", "shipped", "pending", "refunded"}
	for id := 1; id <= 80; id++ {
		total := 0.0
		for k, n := 0, 1+r.next(3); k < n; k++ {
			prod := r.next(len(products.rows))
			qty := 1 + r.next(4)
			total += products.rows[prod][3].(float64) * float64(qty)
			items.rows = append(items.rows, []value{float64(id), float64(prod + 1), float64(qty)})
		}
		orders.rows = append(orders.rows, []value{float64(id), float64(1 + r.next(len(names))), total, statuses[r.next(len(statuses))],
			fmt.Sprintf("2026-%02d-%02d", 1+r.next(10), 1+r.next(28))})
	}
	// A few customers have never ordered, and one order has no status, so
	// joins and NULL have something to show.
	orders.rows[5][3] = nil
	return &database{tables: []*table{customers, orders, products, items}}
}
