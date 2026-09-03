package flight

import "time"

type ID int64
type AirportCode string
type AirlineCode string
type CabinCode string

const (
	CabinEconomy  CabinCode = "Y"
	CabinPremium  CabinCode = "W"
	CabinBusiness CabinCode = "J"
	CabinFirst    CabinCode = "F"
)

type Airport struct {
	ID        int64       `json:"id"`
	Code      AirportCode `json:"code"`
	Name      string      `json:"name"`
	City      string      `json:"city"`
	Country   string      `json:"country"`
	Timezone  string      `json:"timezone"`
	Latitude  *float64    `json:"latitude,omitempty"`
	Longitude *float64    `json:"longitude,omitempty"`
	Status    string      `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
}

type Airline struct {
	ID        int64       `json:"id"`
	Code      AirlineCode `json:"code"`
	Name      string      `json:"name"`
	Country   string      `json:"country"`
	LogoURL   *string     `json:"logo_url,omitempty"`
	Status    string      `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
}

type Route struct {
	ID            int64     `json:"id"`
	AirlineID     int64     `json:"airline_id"`
	OriginID      int64     `json:"origin_id"`
	DestinationID int64     `json:"destination_id"`
	DistanceKM    *float64  `json:"distance_km,omitempty"`
	BasePrice     float64   `json:"base_price"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

type Aircraft struct {
	ID           int64     `json:"id"`
	AirlineID    int64     `json:"airline_id"`
	Model        string    `json:"model"`
	Registration string    `json:"registration"`
	TotalSeats   int       `json:"total_seats"`
	CreatedAt    time.Time `json:"created_at"`
}

type FlightSchedule struct {
	ID            int64     `json:"id"`
	RouteID       int64     `json:"route_id"`
	AircraftID    int64     `json:"aircraft_id"`
	FlightNumber  string    `json:"flight_number"`
	DepartureTime time.Time `json:"departure_time"`
	ArrivalTime   time.Time `json:"arrival_time"`
	DaysOfWeek    int       `json:"days_of_week"`
	ValidFrom     time.Time `json:"valid_from"`
	ValidTo       time.Time `json:"valid_to"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

type FlightInstance struct {
	ID            int64     `json:"id"`
	ScheduleID    int64     `json:"schedule_id"`
	DepartureDate time.Time `json:"departure_date"`
	DepartureTime time.Time `json:"departure_time"`
	ArrivalTime   time.Time `json:"arrival_time"`
	Status        string    `json:"status"`
	DelayMinutes  int       `json:"delay_minutes"`
	CreatedAt     time.Time `json:"created_at"`
}

type CabinClass struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type FareClass struct {
	ID               int64   `json:"id"`
	CabinID          int64   `json:"cabin_id"`
	Code             string  `json:"code"`
	Name             string  `json:"name"`
	BaggageAllowance *string `json:"baggage_allowance,omitempty"`
	ChangeAllowed    bool    `json:"change_allowed"`
	ChangePenalty    float64 `json:"change_penalty"`
	RefundAllowed    bool    `json:"refund_allowed"`
	RefundPenalty    float64 `json:"refund_penalty"`
	UpgradeAllowed   bool    `json:"upgrade_allowed"`
}

type Seat struct {
	ID              int64  `json:"id"`
	AircraftID      int64  `json:"aircraft_id"`
	RowNum          int    `json:"row"`
	Col             string `json:"col"`
	Position        string `json:"position"`
	CabinClassID    int64  `json:"cabin_class_id"`
	IsEmergencyExit bool   `json:"is_emergency_exit"`
	IsExtraLegroom  bool   `json:"is_extra_legroom"`
}

type SearchCriteria struct {
	Origin      AirportCode `json:"origin"`
	Destination AirportCode `json:"destination"`
	Date        time.Time   `json:"date"`
	Passengers  int         `json:"passengers"`
	CabinClass  string      `json:"cabin_class"`
}

type FareOption struct {
	FareClassCode string  `json:"fare_class_code"`
	Cabin         string  `json:"cabin"`
	Available     int     `json:"available"`
	Price         float64 `json:"price"`
	Taxes         float64 `json:"taxes"`
	Total         float64 `json:"total"`
}

type SearchResult struct {
	FlightID     ID           `json:"flight_id"`
	Airline      Airline      `json:"airline"`
	FlightNumber string       `json:"flight_number"`
	Origin       Airport      `json:"origin"`
	Destination  Airport      `json:"destination"`
	Departure    time.Time    `json:"departure"`
	Arrival      time.Time    `json:"arrival"`
	DurationMins int          `json:"duration_minutes"`
	Aircraft     string       `json:"aircraft"`
	Fares        []FareOption `json:"fares"`
}

type SeatMapData struct {
	Aircraft string                `json:"aircraft"`
	Cabins   []CabinSection        `json:"cabins"`
	Seats    []SeatInfo            `json:"seats"`
	Legend   map[string]LegendItem `json:"legend"`
}

type CabinSection struct {
	Code string   `json:"code"`
	Name string   `json:"name"`
	Rows RowRange `json:"rows"`
	Cols []string `json:"cols"`
}

type RowRange struct {
	From int `json:"from"`
	To   int `json:"to"`
}

type SeatInfo struct {
	ID              int64  `json:"id"`
	Row             int    `json:"row"`
	Col             string `json:"col"`
	Position        string `json:"position"`
	Cabin           string `json:"cabin"`
	IsAvailable     bool   `json:"is_available"`
	IsEmergencyExit bool   `json:"is_emergency_exit"`
	IsExtraLegroom  bool   `json:"is_extra_legroom"`
}

type LegendItem struct {
	Color string `json:"color"`
	Label string `json:"label"`
}
