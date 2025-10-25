package health

import (
	json "github.com/json-iterator/go"
	"stream/middleware"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CheckHealth() (map[string]string, error) {
	err := s.repo.Ping()
	if err != nil {
		return nil, err
	}
	return map[string]string{"database": "ok"}, nil
}

func (s *Service) CheckHealthStream() <-chan middleware.StreamChunk {
	chunkChan := make(chan middleware.StreamChunk, 2)
	go func() {
		defer close(chunkChan)
		err := s.repo.Ping()
		if err != nil {
			jsonData, _ := json.Marshal(map[string]string{"database": "error"})
			chunkChan <- middleware.StreamChunk{
				JSONBuf: &jsonData,
				Error:   err,
			}
			return
		}
		jsonData, _ := json.Marshal(map[string]string{"database": "ok"})
		chunkChan <- middleware.StreamChunk{
			JSONBuf: &jsonData,
			Error:   nil,
		}
	}()
	return chunkChan
}
