# High-Throughput SQS Architecture with Java 21 Virtual Threads

## O Problema
Em cenários de alta vazão, abordagens tradicionais de consumo sequencial ou baseadas em threads de plataforma (OS threads) frequentemente resultam em gargalos. O processamento sequencial deixa a CPU ociosa enquanto aguarda I/O (latência de rede, banco de dados, etc.), levando ao acúmulo massivo de mensagens na fila. O modelo tradicional de thread-per-request também sofre com o alto custo de contexto e limitações de memória ao tentar escalar para milhares de conexões simultâneas.

## A Solução
Este projeto implementa uma arquitetura de alta performance utilizando **Java 21 Virtual Threads (Project Loom)** e o **AWS SDK v2**.

### Diferenciais Técnicos:
*   **Virtual Threads:** Utilizamos `Executors.newVirtualThreadPerTaskExecutor()` para disparar uma thread leve para cada mensagem recebida. Isso permite que a aplicação gerencie milhares de tarefas de I/O simultâneas com overhead mínimo, maximizando o uso da CPU e eliminando o bloqueio de threads do sistema operacional.
*   **Consumo em Lote (Batching):** Configuramos o `ReceiveMessageRequest` para buscar até 10 mensagens por requisição, reduzindo drasticamente o número de chamadas de rede (round-trips) e melhorando a eficiência do throughput.
*   **Cliente HTTP Otimizado:** O `SqsClient` é configurado com um `ApacheHttpClient` customizado, permitindo até 5000 conexões simultâneas para evitar gargalos de conexão durante picos de carga.

## Stack Tecnológico
*   **Java 21** (Spring Boot 3.x)
*   **AWS SQS** (Simulado via LocalStack)
*   **Docker & Docker Compose**
*   **Go** (Producer de alta performance)

## Como Executar

### Pré-requisitos
*   Docker e Docker Compose instalados.
*   Permissão de execução no script de inicialização: `chmod +x init-sqs.sh`

### 1. Subindo a Infraestrutura e Escalando Workers
Inicie o LocalStack e escale os consumidores (workers) para processar a carga em paralelo. No exemplo abaixo, subimos 5 instâncias do worker:

```bash
docker compose up -d --scale worker=5
```

### 2. Executando o Producer (Go)
Utilize um container Go efêmero para gerar a carga de mensagens na fila `teste-fila`. O comando abaixo monta o diretório do producer e executa o código Go:

```bash
docker run --rm -it -v "$PWD/sqs-producer":/app -w /app --network host -e AWS_ACCESS_KEY_ID=test -e AWS_SECRET_ACCESS_KEY=test golang:1.23-alpine go run main.go
```

### 3. Monitoramento

#### Monitorar Recursos (CPU/Memória)
Acompanhe o consumo de recursos dos containers para verificar a eficiência das Virtual Threads:

```bash
docker stats
```

#### Monitorar Profundidade da Fila
Utilize o AWS CLI (configurado para o LocalStack) para observar o esvaziamento da fila em tempo real:

```bash
watch -n 1 "aws --endpoint-url=http://localhost:4566 sqs get-queue-attributes --queue-url http://localhost:4566/000000000000/teste-fila --attribute-names ApproximateNumberOfMessages"
```

#### Monitorar Throughput dos Workers
Acompanhe em tempo real a taxa de processamento (mensagens por segundo) de todos os workers ativos:

```bash
watch -n 1 "docker compose ps -q worker | xargs -I {} docker logs --tail 1 {} | grep -o 'Throughput:.*'"
```

## Resultados Observados
*   **Alta Vazão:** Capacidade de processar milhões de mensagens em tempo recorde, limitado apenas pela capacidade de I/O da máquina host.
*   **Eficiência de Recursos:** Baixo consumo de memória e uso eficiente de CPU, sem o overhead de context switching excessivo das threads tradicionais.
*   **Escalabilidade:** A arquitetura permite escalar horizontalmente o número de workers via Docker Compose ou ECS sem alterações no código.
