package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
)

const (
	Region     = "us-east-1"
	Endpoint   = "http://localhost:4566"
	TotalMsgs  = 100000 // Teste com 100k, aumente conforme necessário
	BatchSize  = 10     // Limite do SQS
	MaxWorkers = 50     // Número de goroutines simultâneas
)

func main() {
	// Configuração da fila via variável de ambiente
	queueName := os.Getenv("QUEUE_NAME")
	if queueName == "" {
		queueName = "teste-fila"
	}
	// Constrói a URL da fila dinamicamente
	queueURL := fmt.Sprintf("%s/000000000000/%s", Endpoint, queueName)

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

	fmt.Printf("🚀 Iniciando bombardeio de %d mensagens na fila '%s'...\n", TotalMsgs, queueName)
	fmt.Printf("🔗 URL da Fila: %s\n", queueURL)

	start := time.Now()

	var wg sync.WaitGroup
	// Canal bufferizado para distribuir o trabalho
	jobs := make(chan int, TotalMsgs/BatchSize)

	// Subindo os workers (Goroutines)
	for w := 1; w <= MaxWorkers; w++ {
		wg.Add(1)
		go worker(client, jobs, &wg, queueURL)
	}

	// Enviando os batches para o canal
	// Cada job representa um batch de 10 mensagens
	numBatches := TotalMsgs / BatchSize
	for j := 0; j < numBatches; j++ {
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

func worker(client *sqs.Client, jobs <-chan int, wg *sync.WaitGroup, queueURL string) {
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
			QueueUrl: aws.String(queueURL),
			Entries:  entries,
		})
		if err != nil {
			fmt.Printf("Erro ao enviar batch: %v\n", err)
		}
	}
}
