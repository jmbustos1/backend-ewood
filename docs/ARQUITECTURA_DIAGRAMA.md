# 📊 Diagrama de Arquitectura - Vista Simplificada

## 🎯 Arquitectura Actual del Sistema

```
┌─────────────────────────────────────────────────────────────────┐
│                        DESARROLLO LOCAL                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐         ┌──────────────┐                    │
│  │   Developer  │────────▶│ Source Code  │                    │
│  │    Machine   │         │   (GitHub)   │                    │
│  └──────────────┘         └──────────────┘                    │
│         │                                                       │
│         │ 1. Edita código                                       │
│         │ 2. Crea env.production                                │
│         │ 3. Ejecuta: ./deploy.sh                               │
│         │                                                       │
│         ▼                                                       │
│  ┌─────────────────────────────────────┐                       │
│  │         deploy.sh                   │                       │
│  │  - Build Docker Image               │                       │
│  │  - Push to ECR                      │                       │
│  │  - Upload to S3                     │                       │
│  │  - Send SSM Command                 │                       │
│  └─────────────────────────────────────┘                       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
                            │
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                          AWS CLOUD                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                    STORAGE LAYER                          │  │
│  │                                                            │  │
│  │  ┌──────────────┐              ┌──────────────┐           │  │
│  │  │  S3 Bucket   │              │  ECR Repo   │           │  │
│  │  │              │              │              │           │  │
│  │  │ • docker-    │              │ • ewood-     │           │  │
│  │  │   compose.yml│              │   backend:   │           │  │
│  │  │ • env.       │              │   latest     │           │  │
│  │  │   production │              │              │           │  │
│  │  └──────────────┘              └──────────────┘           │  │
│  │                                                            │  │
│  └──────────────────────────────────────────────────────────┘  │
│                            │                                     │
│                            │ Download                            │
│                            ▼                                     │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                    COMPUTE LAYER                          │  │
│  │                                                            │  │
│  │  ┌────────────────────────────────────────────────────┐   │  │
│  │  │           EC2 Instance (Ubuntu 22.04)             │   │  │
│  │  │  ┌──────────────────────────────────────────────┐  │   │  │
│  │  │  │  Docker Engine                              │  │   │  │
│  │  │  │  ┌──────────────────────────────────────┐  │  │   │  │
│  │  │  │  │  Container: ewood-golang-container   │  │  │   │  │
│  │  │  │  │  - /app/server (Go binary)           │  │  │   │  │
│  │  │  │  │  - /app/migrate (migration tool)     │  │  │   │  │
│  │  │  │  │  - /app/migrations/ (SQL files)      │  │  │   │  │
│  │  │  │  │  - Port: 8080 → 8200 (host)          │  │  │   │  │
│  │  │  │  │  - env.production (from S3)          │  │  │   │  │
│  │  │  │  └──────────────────────────────────────┘  │  │   │  │
│  │  │  └──────────────────────────────────────────────┘  │  │   │  │
│  │  │                                                     │  │   │  │
│  │  │  ┌──────────────────────────────────────────────┐  │  │   │  │
│  │  │  │  SSM Agent                                   │  │  │   │  │
│  │  │  │  - Recibe comandos de despliegue             │  │  │   │  │
│  │  │  │  - Ejecuta scripts                           │  │  │   │  │
│  │  │  └──────────────────────────────────────────────┘  │  │   │  │
│  │  └────────────────────────────────────────────────────┘   │  │   │  │
│  │                                                            │  │   │  │
│  └──────────────────────────────────────────────────────────┘  │  │   │  │
│                            │                                     │  │   │  │
│                            │ Connect                             │  │   │  │
│                            ▼                                     │  │   │  │
│  ┌──────────────────────────────────────────────────────────┐  │  │   │  │
│  │                    DATABASE LAYER                        │  │  │   │  │
│  │                                                            │  │  │   │  │
│  │  ┌────────────────────────────────────────────────────┐  │  │   │  │
│  │  │  RDS PostgreSQL                                     │  │  │   │  │
│  │  │  - Instance: db.t3.micro                            │  │  │   │  │
│  │  │  - Storage: 20 GB (gp3)                            │  │  │   │  │
│  │  │  - Engine: PostgreSQL 16                           │  │  │   │  │
│  │  │  - Port: 5432                                       │  │  │   │  │
│  │  │  - Database: ewood_db                              │  │  │   │  │
│  │  │  - Tables: users, migrations_log, schema_migrations│  │  │   │  │
│  │  └────────────────────────────────────────────────────┘  │  │   │  │
│  │                                                            │  │   │  │
│  └──────────────────────────────────────────────────────────┘  │  │   │  │
│                                                                  │  │   │  │
└─────────────────────────────────────────────────────────────────┘  │  │   │  │
                                                                      │  │   │  │
                            ┌─────────────────────────────────────────┘  │  │  │
                            │                                            │  │  │
                            │ External Access                            │  │  │
                            ▼                                            │  │  │
┌─────────────────────────────────────────────────────────────────┐      │  │  │
│                    EXTERNAL ACCESS                             │      │  │  │
│                                                                 │      │  │  │
│  ┌──────────────┐         ┌──────────────┐                    │      │  │  │
│  │   Browser/   │────────▶│  EC2:8200    │                    │      │  │  │
│  │   API Client │         │  /health     │                    │      │  │  │
│  └──────────────┘         └──────────────┘                    │      │  │  │
│                                                                 │      │  │  │
│  ┌──────────────┐         ┌──────────────┐                    │      │  │  │
│  │   DBeaver    │────────▶│  RDS:5432    │                    │      │  │  │
│  │   (DB Tool)  │         │  PostgreSQL  │                    │      │  │  │
│  └──────────────┘         └──────────────┘                    │      │  │  │
│                                                                 │      │  │  │
└─────────────────────────────────────────────────────────────────┘      │  │  │
                                                                          │  │  │
┌─────────────────────────────────────────────────────────────────┐      │  │  │
│                    NETWORKING                                   │      │  │  │
│                                                                 │      │  │  │
│  ┌──────────────┐         ┌──────────────┐                    │      │  │  │
│  │ Security     │         │ Security     │                    │      │  │  │
│  │ Group EC2    │         │ Group RDS     │                    │      │  │  │
│  │              │         │              │                    │      │  │  │
│  │ In: 8200     │         │ In: 5432     │                    │      │  │  │
│  │     (0.0.0.0)│         │     (EC2 SG)│                    │      │  │  │
│  │     (0.0.0.0)│         │     (0.0.0.0)│                    │      │  │  │
│  └──────────────┘         └──────────────┘                    │      │  │  │
│                                                                 │      │  │  │
└─────────────────────────────────────────────────────────────────┘      │  │  │
                                                                          │  │  │
┌─────────────────────────────────────────────────────────────────┐      │  │  │
│                    IAM & PERMISSIONS                             │      │  │  │
│                                                                 │      │  │  │
│  EC2 IAM Role: ec2-ssm-role-golang-ewood                        │      │  │  │
│  ├─ s3:GetObject (leer S3)                                      │      │  │  │
│  ├─ ecr:GetAuthorizationToken (login ECR)                      │      │  │  │
│  ├─ ecr:BatchGetImage (pull images)                             │      │  │  │
│  └─ ssm:SendCommand (recibir comandos)                          │      │  │  │
│                                                                 │      │  │  │
└─────────────────────────────────────────────────────────────────┘      │  │  │
                                                                          │  │  │
```

---

## 📝 Flujo de Datos (Dónde se Guarda Cada Cosa)

### 1. Código Fuente
```
📍 Ubicación: Git Repository (GitHub/GitLab)
📦 Contenido: 
   - cmd/server/main.go
   - internal/handler/*.go
   - migrations/*.sql
   - Dockerfile
   - docker-compose.yml
```

### 2. Variables de Despliegue
```
📍 Ubicación: .env.push (local, NO commiteado)
📦 Contenido:
   - AWS_ACCOUNT_ID
   - EC2_INSTANCE_ID
   - S3_BUCKET_NAME
   - ECR_REPOSITORY
   - AWS_REGION
```

### 3. Variables de Runtime (Secrets)
```
📍 Ubicación ACTUAL: 
   Local → S3 → EC2 → Container

📦 Contenido (env.production):
   - POSTGRES_HOST
   - POSTGRES_PASSWORD ⚠️
   - POSTGRES_USER
   - POSTGRES_DB
   - MONGODB_URI ⚠️ (futuro)

📍 Ubicación RECOMENDADA (Producción):
   AWS Secrets Manager → EC2 → Container
```

### 4. Docker Compose
```
📍 Ubicación: 
   Local → S3 → EC2

📦 Contenido (docker-compose.yml):
   - Image: ECR URI
   - Ports: 8200:8080
   - env_file: /home/ubuntu/env.production
```

### 5. Docker Image
```
📍 Ubicación: ECR (Elastic Container Registry)
📦 Contenido:
   - /app/server (binario Go)
   - /app/migrate (binario migraciones)
   - /app/migrations/ (archivos SQL)
```

### 6. Base de Datos
```
📍 Ubicación: RDS PostgreSQL
📦 Contenido:
   - Tablas creadas por migraciones
   - Datos de la aplicación
   - schema_migrations (tracking)
```

---

## 🔄 Flujo Completo de Despliegue

```
┌─────────────────────────────────────────────────────────────┐
│ PASO 1: PREPARACIÓN LOCAL                                   │
└─────────────────────────────────────────────────────────────┘
Developer Machine
├─ Edita código
├─ Crea/actualiza env.production
└─ Ejecuta: source .env.push && ./deploy.sh

┌─────────────────────────────────────────────────────────────┐
│ PASO 2: BUILD & PUSH                                        │
└─────────────────────────────────────────────────────────────┘
Local → ECR
├─ docker build (compila código)
├─ docker tag (etiqueta imagen)
└─ docker push (sube a ECR)

┌─────────────────────────────────────────────────────────────┐
│ PASO 3: UPLOAD CONFIG                                        │
└─────────────────────────────────────────────────────────────┘
Local → S3
├─ aws s3 cp docker-compose.yml → S3
└─ aws s3 cp env.production → S3

┌─────────────────────────────────────────────────────────────┐
│ PASO 4: SSM COMMAND                                         │
└─────────────────────────────────────────────────────────────┘
Local → AWS SSM → EC2
└─ aws ssm send-command (envía script de despliegue)

┌─────────────────────────────────────────────────────────────┐
│ PASO 5: DEPLOY EN EC2                                       │
└─────────────────────────────────────────────────────────────┘
EC2 Instance (via SSM)
├─ Descarga docker-compose.yml desde S3
├─ Descarga env.production desde S3
├─ Login ECR
├─ Pull image desde ECR
├─ docker compose down (stop old)
├─ docker compose up -d (start new)
└─ Ejecuta migraciones

┌─────────────────────────────────────────────────────────────┐
│ PASO 6: RUNTIME                                             │
└─────────────────────────────────────────────────────────────┘
Container en EC2
├─ Lee env.production
├─ Conecta a RDS
└─ Inicia servidor HTTP
```

---

## 🎯 Resumen Ejecutivo

| Componente | Dónde se Guarda | Cómo se Mueve |
|------------|-----------------|---------------|
| **Código** | Git | Push/Pull |
| **Imagen Docker** | ECR | Push/Pull |
| **docker-compose.yml** | S3 | Upload/Download |
| **env.production** | S3 | Upload/Download |
| **Migrations** | Docker Image | Incluidas en imagen |
| **Datos** | RDS | Persistente |

