# Transit API JP

Go REST API for Transit (trains/buses) in Japan

Uses NAVITIME API via RAPIDAPI

## Transit

Calls go transit enpoint with start, goal and start date. Return all data from those two point.

### JA

`/transit?start={station_name}&goal={station_name}&start_time={start_time} (datetime format: 2020-08-19T10%3A00%3A00)`

### EN

Need to add `lang=en` param to get english names

`/transit?lang=en&start={station_name}&goal={station_name}&start_time={start_time} (datetime format: 2020-08-19T10%3A00%3A00)`

Tokenize the kanji to kana then convert the kana to romaji

## Autocomplete

Returns a list of objects for stations based on input using `word` param

### JA

`/autocomplete?word=station_name`

### EN

Need to add `lang=en` param to get english names

`/autocomplete?lang=en&word=station_name`

Tokenize the kanji to kana then convert the kana to romaji
