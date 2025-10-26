package health

import (
	"stream/middleware"

	json "github.com/json-iterator/go"
)

type Service struct {
	dummyRepo *Repository
	realRepo  *Repository
}

func NewService(dummyRepo *Repository, realRepo *Repository) *Service {
	return &Service{
		dummyRepo: dummyRepo,
		realRepo:  realRepo,
	}
}

func (s *Service) CheckHealth() (map[string]string, error) {
	result := make(map[string]string)

	// Check dummy database
	err := s.dummyRepo.Ping()
	if err != nil {
		result["dummy_database"] = "error"
	} else {
		result["dummy_database"] = "ok"
	}

	// Check real database
	err = s.realRepo.Ping()
	if err != nil {
		result["real_database"] = "error"
	} else {
		result["real_database"] = "ok"
	}

	return result, nil
}

func (s *Service) CheckHealthStream() <-chan middleware.StreamChunk {
	chunkChan := make(chan middleware.StreamChunk, 2)
	go func() {
		defer close(chunkChan)

		result := make(map[string]string)

		// Check dummy database
		err := s.dummyRepo.Ping()
		if err != nil {
			result["dummy_database"] = "error"
		} else {
			result["dummy_database"] = "ok"
		}

		// Check real database
		err = s.realRepo.Ping()
		if err != nil {
			result["real_database"] = "error"
		} else {
			result["real_database"] = "ok"
		}

		jsonData, _ := json.Marshal(result)
		chunkChan <- middleware.StreamChunk{
			JSONBuf: &jsonData,
			Error:   nil,
		}
	}()
	return chunkChan
}
