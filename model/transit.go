package model

import "time"

// TransitResponse represents the top-level response structure
type TransitResponse struct {
	Items []TransitItem `json:"items"`
	Unit  Unit          `json:"unit"`
}

// TransitItem represents a single transit route option
type TransitItem struct {
	Summary  Summary   `json:"summary"`
	Sections []Section `json:"sections"`
}

// Summary contains the overview of the transit route
type Summary struct {
	No    string `json:"no"`
	Start Point  `json:"start"`
	Goal  Point  `json:"goal"`
	Move  Move   `json:"move"`
}

// Point represents a location (station or place)
type Point struct {
	Type      string     `json:"type"`
	Coord     Coordinate `json:"coord"`
	Name      string     `json:"name"`
	NodeID    string     `json:"node_id"`
	NodeTypes []string   `json:"node_types"`
	Numbering *Numbering `json:"numbering,omitempty"`
}

// Coordinate represents latitude and longitude
type Coordinate struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// Numbering represents station numbering information
type Numbering struct {
	Departure []StationNumber `json:"departure,omitempty"`
	Arrival   []StationNumber `json:"arrival,omitempty"`
}

// StationNumber represents station number with symbol
type StationNumber struct {
	Symbol string `json:"symbol"`
	Number string `json:"number"`
}

// Move represents movement details for a route
type Move struct {
	TransitCount int       `json:"transit_count"`
	Fare         Fare      `json:"fare"`
	Type         string    `json:"type"`
	FromTime     time.Time `json:"from_time"`
	ToTime       time.Time `json:"to_time"`
	Time         int       `json:"time"`
	Distance     int       `json:"distance"`
	MoveType     []string  `json:"move_type"`
}

// Fare represents fare information for different ticket types
type Fare struct {
	Unit0        float64 `json:"unit_0"`
	Unit48       float64 `json:"unit_48"`
	Unit128Train float64 `json:"unit_128_train,omitempty"`
	Unit130Train float64 `json:"unit_130_train,omitempty"`
	Unit133Train float64 `json:"unit_133_train,omitempty"`
	Unit128      float64 `json:"unit_128,omitempty"`
	Unit130      float64 `json:"unit_130,omitempty"`
	Unit133      float64 `json:"unit_133,omitempty"`
	Unit136      float64 `json:"unit_136,omitempty"`
	Unit138      float64 `json:"unit_138,omitempty"`
	Unit141      float64 `json:"unit_141,omitempty"`
}

// Section represents either a point or a move section in the route
type Section struct {
	// Common fields
	Type string `json:"type"`

	// Point fields
	Coord     *Coordinate `json:"coord,omitempty"`
	Name      string      `json:"name,omitempty"`
	NodeID    string      `json:"node_id,omitempty"`
	NodeTypes []string    `json:"node_types,omitempty"`
	Numbering *Numbering  `json:"numbering,omitempty"`

	// Move fields
	Transport *Transport `json:"transport,omitempty"`
	Move      string     `json:"move,omitempty"`
	FromTime  *time.Time `json:"from_time,omitempty"`
	ToTime    *time.Time `json:"to_time,omitempty"`
	Time      int        `json:"time,omitempty"`
	Distance  int        `json:"distance,omitempty"`
	LineName  string     `json:"line_name,omitempty"`
}

// Transport represents transportation details
type Transport struct {
	Fare       Fare         `json:"fare"`
	Color      string       `json:"color"`
	Name       string       `json:"name"`
	FareSeason string       `json:"fare_season"`
	Company    Company      `json:"company"`
	Links      []Link       `json:"links"`
	ID         string       `json:"id"`
	Type       string       `json:"type"`
	FareBreak  FareBreak    `json:"fare_break"`
	FareDetail []FareDetail `json:"fare_detail"`
}

// Company represents transportation company information
type Company struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Link represents route link information
type Link struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Direction   string      `json:"direction"`
	Destination Destination `json:"destination"`
	From        Station     `json:"from"`
	To          Station     `json:"to"`
	IsTimetable string      `json:"is_timetable"`
}

// Destination represents link destination
type Destination struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Station represents a station in a link
type Station struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// FareBreak represents fare break information
type FareBreak struct {
	Unit0   bool `json:"unit_0"`
	Unit48  bool `json:"unit_48"`
	Unit128 bool `json:"unit_128"`
	Unit130 bool `json:"unit_130"`
	Unit133 bool `json:"unit_133"`
	Unit136 bool `json:"unit_136"`
	Unit138 bool `json:"unit_138"`
	Unit141 bool `json:"unit_141"`
}

// FareDetail represents detailed fare information
type FareDetail struct {
	Start Station `json:"start"`
	ID    string  `json:"id"`
	Goal  Station `json:"goal"`
	Fare  float64 `json:"fare"`
}

// Unit represents units used in the response
type Unit struct {
	Datum     string `json:"datum"`
	CoordUnit string `json:"coord_unit"`
	Distance  string `json:"distance"`
	Time      string `json:"time"`
	Currency  string `json:"currency"`
}

// NodeResponse represents the response from the transport_node API
type NodeResponse struct {
	Items []NodeItem `json:"items"`
}

// NodeItem represents a single node item
type NodeItem struct {
	ID string `json:"id"`
}

// AutocompleteResponse represents the response from the autocomplete API
type AutocompleteResponse struct {
	Items []AutocompleteStation `json:"items"`
}

// AutocompleteStation represents a station in autocomplete response
type AutocompleteStation struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Ruby        string          `json:"ruby"`
	Types       []string        `json:"types"`
	AddressName string          `json:"address_name"`
	AddressCode string          `json:"address_code"`
	Coord       Coordinate      `json:"coord"`
	Numbering   []StationNumber `json:"numbering"`
	Type        string          `json:"type"` // This field will store the first type
}

// FilteredStation represents a filtered station in autocomplete response
type FilteredStation struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	Ruby string `json:"ruby,omitempty"`
}

// FilteredAutocompleteResponse represents the filtered autocomplete response
type FilteredAutocompleteResponse struct {
	Items []FilteredStation `json:"items"`
}
