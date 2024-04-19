package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const API = "http://ip-api.com/json"

type Semaphore chan struct{}

func NewSemaphore(max int) Semaphore {
	return make(chan struct{}, max)
}

func (s Semaphore) Acquire() {
	s <- struct{}{}
}

func (s Semaphore) Release() {
	<-s
}

func main() {
	urls := []string{"201.122.10.1", "24.48.0.1"}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Define el máximo de gorutines concurrentes
	maxGorutines := 5

	// Crea un canal para limitar el número de goroutines
	// semaphore := make(chan struct{}, maxGorutines)
	semaphore := NewSemaphore(maxGorutines)

	// Canal para los resultados y errores
	results := make(chan *Location, len(urls))
	errors := make(chan error, len(urls))
	var wg sync.WaitGroup

	for _, url := range urls {
		// Añade una goroutine al grupo de espera
		wg.Add(1)

		// Intenta obtener un semáforo
		semaphore.Acquire()

		go func(ip string) {
			// Defer la elimina del semáforo después de terminar
			defer func() { semaphore.Release() }()

			defer wg.Done()

			ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()

			// Llamar api por json
			data, err := GetLocation(ctx, ip)
			if err != nil {
				errors <- err
			}

			results <- data
		}(url)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	/*
		La utilización de select sería más apropiada si tuviéramos múltiples canales
		que queremos leer o si necesitáramos realizar otras operaciones simultáneas,
		como la cancelación o el tiempo de espera.
	*/

	var resultsClosed, errorsClosed bool
	for !resultsClosed || errorsClosed {
		select {
		case loc, ok := <-results:
			if !ok {
				resultsClosed = true
			} else {
				fmt.Printf("Location lat:%v, lon:%v, City:%s\n", loc.Lat, loc.Lon, loc.City)
			}
		case err, ok := <-errors:
			if !ok {
				errorsClosed = true
			} else {
				fmt.Printf("Error: %s\n", err)
			}
		}
	}
}

func GetLocation(ctx context.Context, url string) (*Location, error) {
	result, err := Request(ctx, url)
	if err != nil {
		return nil, err
	}

	var location *Location
	if err := json.Unmarshal(result, &location); err != nil {
		return nil, err
	}

	return location, nil
}

func Request(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/%s", API, url), nil)
	if err != nil {
		return nil, err
	}

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

type Location struct {
	Query         string  `json:"query,omitempty"`
	Status        string  `json:"status,omitempty"`
	Continent     string  `json:"continent,omitempty"`
	ContinentCode string  `json:"continentCode,omitempty"`
	Country       string  `json:"country,omitempty"`
	CountryCode   string  `json:"countryCode,omitempty"`
	Region        string  `json:"region,omitempty"`
	RegionName    string  `json:"regionName,omitempty"`
	City          string  `json:"city,omitempty"`
	District      string  `json:"district,omitempty"`
	Zip           string  `json:"zip,omitempty"`
	Lat           float64 `json:"lat,omitempty"`
	Lon           float64 `json:"lon,omitempty"`
	Timezone      string  `json:"timezone,omitempty"`
	Offset        int     `json:"offset,omitempty"`
	Currency      string  `json:"currency,omitempty"`
	Isp           string  `json:"isp,omitempty"`
	Org           string  `json:"org,omitempty"`
	As            string  `json:"as,omitempty"`
	Asname        string  `json:"asname,omitempty"`
	Mobile        bool    `json:"mobile,omitempty"`
	Proxy         bool    `json:"proxy,omitempty"`
	Hosting       bool    `json:"hosting,omitempty"`
}
