package main

// sampleJSON is what the explorer opens when it is given no file: a made-up
// API response with nesting, arrays, every kind of value and a key that needs
// quoting in a path.
const sampleJSON = `{
  "status": "ok",
  "generated": "2026-10-18T09:30:00Z",
  "shop": {"name": "Analytical Engines", "open": true, "rating": 4.50, "closed_on": null},
  "orders": [
    {
      "id": 1001,
      "customer": {"name": "Ada Lovelace", "email": "ada@example.com", "tags": ["vip", "early"]},
      "lines": [
        {"sku": "GEAR-12", "qty": 4, "price": 12.50},
        {"sku": "CARD-7", "qty": 100, "price": 0.15}
      ],
      "paid": true,
      "note": "Deliver to the back door; \"ring twice\""
    },
    {
      "id": 1002,
      "customer": {"name": "Charles Babbage", "email": "charles@example.com", "tags": []},
      "lines": [{"sku": "ENGINE-1", "qty": 1, "price": 99999.99}],
      "paid": false,
      "note": null
    }
  ],
  "totals": {"orders": 2, "revenue": 100062.49, "by-status": {"paid": 1, "unpaid": 1}}
}
`
