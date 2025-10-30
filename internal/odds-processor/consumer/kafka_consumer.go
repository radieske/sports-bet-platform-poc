package consumer

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/radieske/sports-bet-platform-poc/internal/odds-processor/cache"
	"github.com/radieske/sports-bet-platform-poc/internal/odds-processor/repository"
	"github.com/radieske/sports-bet-platform-poc/pkg/contracts/events"
)

// Processor consome mensagens de odds do Kafka, faz cache e persiste no banco
// Callbacks de métricas podem ser usadas para monitoramento de cada etapa
type Processor struct {
	Log    *zap.Logger
	Reader *kafka.Reader
	Repo   *repository.PostgresRepo
	Cache  *cache.RedisCache

	OnConsumed func()       // métricas (counter++)
	OnCached   func()       // métricas
	OnPersist  func()       // métricas
	OnError    func(string) // métricas por fase
}

// Run inicia o loop principal de consumo e processamento das mensagens Kafka
func (p *Processor) Run(ctx context.Context) error {
	for {
		m, err := p.Reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err() // encerra se o contexto for cancelado
			}
			p.Log.Warn("kafka read failed", zap.Error(err))
			if p.OnError != nil {
				p.OnError("read")
			}
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if p.OnConsumed != nil {
			p.OnConsumed() // callback de métrica: mensagem consumida
		}

		var ev events.OddsUpdate
		if err := json.Unmarshal(m.Value, &ev); err != nil {
			p.Log.Warn("invalid message", zap.Error(err))
			if p.OnError != nil {
				p.OnError("decode")
			}
			continue
		}

		// Atualiza cache Redis com a odd atual
		if err := p.Cache.SetCurrent(ctx, ev); err != nil {
			p.Log.Warn("redis set failed", zap.Error(err))
			if p.OnError != nil {
				p.OnError("cache")
			}
			// não bloqueia persistência se falhar o cache
		} else if p.OnCached != nil {
			p.OnCached() // callback de métrica: cache atualizado
		}

		// Persiste/atualiza odd atual e histórico no Postgres
		if err := p.Repo.UpsertCurrent(ctx, ev); err != nil {
			p.Log.Warn("db upsert failed", zap.Error(err))
			if p.OnError != nil {
				p.OnError("db_upsert")
			}
			continue
		}
		if err := p.Repo.InsertHistory(ctx, ev); err != nil {
			p.Log.Warn("db insert history failed", zap.Error(err))
			if p.OnError != nil {
				p.OnError("db_history")
			}
			continue
		}
		if p.OnPersist != nil {
			p.OnPersist() // callback de métrica: persistência concluída
		}
	}
}
