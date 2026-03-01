# Estágio 1: Build
FROM gradle:8.6-jdk21-alpine AS build
COPY --chown=gradle:gradle . /home/gradle/src
WORKDIR /home/gradle/src
# Compila o projeto e gera o shadowJar ou o jar padrão
RUN gradle build --no-daemon -x test

# Estágio 2: Runtime (JRE 21)
FROM eclipse-temurin:21-jre-alpine
WORKDIR /app

# Copia o JAR do estágio de build (ajuste o nome se seu jar tiver outro padrão)
COPY --from=build /home/gradle/src/build/libs/*.jar app.jar

# Variáveis de ambiente que a sua app vai ler
ENV SQS_ENDPOINT=http://localstack:4566
ENV QUEUE_URL=http://localstack:4566/000000000000/teste-fila
ENV AWS_REGION=us-east-1

ENTRYPOINT ["java", "-jar", "app.jar"]