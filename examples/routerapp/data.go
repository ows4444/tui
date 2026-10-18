package main

import "fmt"

type customer struct {
	id         int
	name, city string
}

type order struct {
	id, cust, cents int
	item            string
}

var customers = []customer{
	{1, "Ada Lovelace", "London"}, {2, "Grace Hopper", "New York"}, {3, "Alan Turing", "Manchester"}, {4, "Katherine Johnson", "Hampton"},
	{5, "Edsger Dijkstra", "Eindhoven"}, {6, "Margaret Hamilton", "Boston"}, {7, "Donald Knuth", "Stanford"}, {8, "Barbara Liskov", "Cambridge"},
	{9, "Dennis Ritchie", "Newark"}, {10, "Radia Perlman", "Portsmouth"}, {11, "Ken Thompson", "New Orleans"}, {12, "Frances Allen", "Peru"},
}

var items = []struct {
	name  string
	cents int
}{{"Keyboard", 8900}, {"Monitor", 21900}, {"USB cable", 900}, {"Desk lamp", 3400}, {"Notebook", 600}, {"Headphones", 12900}}

// orders are made from a fixed rule, so every run has the same ones: customer
// c has three, numbered from 1001.
var orders = func() []order {
	var out []order
	for c := range customers {
		for k := 0; k < 3; k++ {
			it := items[(c*2+k*3)%len(items)]
			out = append(out, order{id: 1001 + len(out), cust: c + 1, cents: it.cents * (1 + (c+k)%3), item: it.name})
		}
	}
	return out
}()

func customerByID(id int) customer { return customers[id-1] }

func orderByID(id int) order { return orders[id-1001] }

func ordersOf(cust int) []order {
	var out []order
	for _, o := range orders {
		if o.cust == cust {
			out = append(out, o)
		}
	}
	return out
}

func money(cents int) string { return fmt.Sprintf("$%d.%02d", cents/100, cents%100) }
