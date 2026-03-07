package models

type TrainResponse struct {
	Data struct {
		Trains []Train `json:"trains"`
	} `json:"data"`
}

type Train struct {
	TripNumber    string     `json:"trip_number"`
	DepartureTime string     `json:"departure_date_time"`
	ArrivalTime   string     `json:"arrival_date_time"`
	SeatTypes     []SeatType `json:"seat_types"`
}

type SeatType struct {
	Type       string `json:"type"`
	Fare       string `json:"fare"`
	SeatCounts struct {
		Online  int `json:"online"`
		Offline int `json:"offline"`
	} `json:"seat_counts"`
}
