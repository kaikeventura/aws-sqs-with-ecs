package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
)

const (
	Region       = "us-east-1"
	QueueURL     = "http://localhost:4566/000000000000/teste-fila"
	Endpoint     = "http://localhost:4566"
	TotalMsgs    = 100000 // Teste com 100k, aumente conforme necessário
	BatchSize    = 10     // Limite do SQS
	MaxWorkers   = 50     // Número de goroutines simultâneas
)

func main() {
	// Configuração do SDK apontando para o LocalStack
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:           Endpoint,
			SigningRegion: Region,
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(Region),
		config.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		log.Fatalf("Erro ao carregar config: %v", err)
	}

	client := sqs.NewFromConfig(cfg)

	fmt.Printf("🚀 Iniciando bombardeio de %d mensagens...\n", TotalMsgs)
	start := time.Now()

	var wg sync.WaitGroup
	jobs := make(chan int, TotalMsgs/BatchSize)

	// Subindo os workers (Goroutines)
	for w := 1; w <= MaxWorkers; w++ {
		wg.Add(1)
		go worker(client, jobs, &wg)
	}

	// Enviando os batches para o canal
	for j := 0; j < TotalMsgs/BatchSize; j++ {
		jobs <- j
	}
	close(jobs)

	wg.Wait()

	duration := time.Since(start)
	throughput := float64(TotalMsgs) / duration.Seconds()

	fmt.Printf("\n✅ Concluído!\n")
	fmt.Printf("⏱️ Tempo total: %v\n", duration)
	fmt.Printf("📈 Vazão: %.2f mensagens/segundo\n", throughput)
}

func worker(client *sqs.Client, jobs <-chan int, wg *sync.WaitGroup) {
	defer wg.Done()
	for range jobs {
		var entries []types.SendMessageBatchRequestEntry
		for i := 0; i < BatchSize; i++ {
			id := uuid.New().String()
			entries = append(entries, types.SendMessageBatchRequestEntry{
				Id:          aws.String(id),
				MessageBody: aws.String(fmt.Sprintf("Payload de teste - %s", id)),
			})
		}

		_, err := client.SendMessageBatch(context.TODO(), &sqs.SendMessageBatchInput{
			QueueUrl: aws.String(QueueURL),
			Entries:  entries,
		})
		if err != nil {
			fmt.Printf("Erro ao enviar batch: %v\n", err)
		}
	}
}