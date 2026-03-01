package com.kaikeventura.awssqswithecs;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.CommandLineRunner;
import org.springframework.context.annotation.Profile;
import org.springframework.stereotype.Component;
import software.amazon.awssdk.auth.credentials.AwsBasicCredentials;
import software.amazon.awssdk.auth.credentials.StaticCredentialsProvider;
import software.amazon.awssdk.http.apache.ApacheHttpClient;
import software.amazon.awssdk.regions.Region;
import software.amazon.awssdk.services.sqs.SqsClient;
import software.amazon.awssdk.services.sqs.model.DeleteMessageRequest;
import software.amazon.awssdk.services.sqs.model.Message;
import software.amazon.awssdk.services.sqs.model.ReceiveMessageRequest;

import java.net.URI;
import java.time.Duration;
import java.util.List;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;

@Component
@Profile({"virtual", "!simple"})
public class SqsConsumerVirtual implements CommandLineRunner {

    private static final Logger log = LoggerFactory.getLogger(SqsConsumerVirtual.class);

    private final String queueUrl;
    private final String endpointUrl;
    private final AtomicInteger processedMessagesTotal = new AtomicInteger(0);
    private final AtomicInteger processedMessagesLastSecond = new AtomicInteger(0);

    public SqsConsumerVirtual(
            @Value("${QUEUE_URL:http://localhost:4566/000000000000/teste-fila}") String queueUrl,
            @Value("${SQS_ENDPOINT:http://localhost:4566}") String endpointUrl
    ) {
        this.queueUrl = queueUrl;
        this.endpointUrl = endpointUrl;
    }

    @Override
    public void run(String... args) {
        SqsClient sqsClient = buildSqsClient();
        
        // Virtual Threads (Java 21) para alta concorrência
        ExecutorService virtualThreadExecutor = Executors.newVirtualThreadPerTaskExecutor();
        
        // Monitoramento de throughput
        ScheduledExecutorService monitorExecutor = Executors.newSingleThreadScheduledExecutor();
        monitorExecutor.scheduleAtFixedRate(() -> {
            int count = processedMessagesLastSecond.getAndSet(0);
            log.info("Throughput: {} msgs/sec | Total: {}", count, processedMessagesTotal.get());
        }, 1, 1, TimeUnit.SECONDS);

        log.info("Starting SQS Consumer (Virtual Threads)...");
        log.info("SQS Endpoint: {}", endpointUrl);
        log.info("Queue URL:    {}", queueUrl);

        // Loop infinito de consumo
        while (true) {
            try {
                ReceiveMessageRequest receiveMessageRequest = ReceiveMessageRequest.builder()
                        .queueUrl(queueUrl)
                        .maxNumberOfMessages(10)
                        .waitTimeSeconds(20)
                        .build();

                List<Message> messages = sqsClient.receiveMessage(receiveMessageRequest).messages();

                for (Message message : messages) {
                    virtualThreadExecutor.submit(() -> {
                        try {
                            processMessage(message);
                            deleteMessage(sqsClient, message);
                            processedMessagesTotal.incrementAndGet();
                            processedMessagesLastSecond.incrementAndGet();
                        } catch (Exception e) {
                            log.error("Error processing message {}: {}", message.messageId(), e.getMessage());
                        }
                    });
                }
            } catch (Exception e) {
                log.error("Error receiving messages: {}", e.getMessage());
                try {
                    Thread.sleep(1000);
                } catch (InterruptedException ie) {
                    Thread.currentThread().interrupt();
                    break;
                }
            }
        }
    }

    private SqsClient buildSqsClient() {
        // Configuração otimizada do Apache HttpClient para alta concorrência
        ApacheHttpClient.Builder httpClientBuilder = ApacheHttpClient.builder()
                .maxConnections(5000) // Aumenta drasticamente o limite de conexões
                .connectionAcquisitionTimeout(Duration.ofSeconds(10)); // Timeout para adquirir conexão

        return SqsClient.builder()
                   .endpointOverride(URI.create(endpointUrl))
                   .region(Region.US_EAST_1)
                   .credentialsProvider(StaticCredentialsProvider.create(AwsBasicCredentials.create("test", "test")))
                   .httpClientBuilder(httpClientBuilder)
                   .build();
    }

    private void processMessage(Message message) {
        try {
            // Simula processamento de 30ms
            Thread.sleep(30);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        }
    }

    private void deleteMessage(SqsClient sqsClient, Message message) {
        DeleteMessageRequest deleteMessageRequest = DeleteMessageRequest.builder()
                .queueUrl(queueUrl)
                .receiptHandle(message.receiptHandle())
                .build();
        sqsClient.deleteMessage(deleteMessageRequest);
    }
}
