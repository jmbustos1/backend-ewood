# 🚀 Flujo de Despliegue - Diagrama Visual

## 📊 Flujo Completo de Despliegue (Versión Visual)

```mermaid
flowchart TB
    subgraph DEV["👨‍💻 DESARROLLADOR"]
        CODE["📝 Código Fuente<br/>GitHub"]
        ENV["🔐 env.production<br/>Archivo Local"]
        DEPLOY["▶️ Ejecuta deploy.sh"]
    end

    subgraph AWS["☁️ AWS CLOUD"]
        subgraph STORAGE["📦 ALMACENAMIENTO"]
            S3["🪣 S3 Bucket<br/>ewood-docker-compose-files<br/><br/>📄 docker-compose.yml<br/>🔐 env.production"]
            ECR["🐳 ECR Repository<br/>ewood-backend<br/><br/>📦 Imagen Docker<br/>con código compilado"]
        end

        subgraph COMPUTE["🖥️ SERVIDOR"]
            EC2["🖥️ EC2 Instance<br/>Ubuntu Server"]
            DOCKER["🐳 Docker Engine"]
            CONTAINER["📦 Contenedor<br/>ewood-golang-container<br/><br/>🌐 Puerto 8200<br/>✅ Aplicación corriendo"]
        end

        subgraph DATABASE["💾 BASE DE DATOS"]
            RDS["🗄️ RDS PostgreSQL<br/>ewood-postgres<br/><br/>📊 Tablas creadas<br/>💾 Datos guardados"]
        end

        subgraph MANAGEMENT["⚙️ GESTIÓN"]
            SSM["📡 SSM Agent<br/>Sistema de Control"]
            IAM["🔑 IAM Role<br/>Permisos"]
        end
    end

    subgraph EXTERNAL["🌍 ACCESO EXTERNO"]
        USER["👤 Usuario Final"]
        DBA["👨‍💼 Administrador DB"]
    end

    %% Desarrollo
    DEV -->|1. Edita y guarda| CODE
    DEV -->|2. Configura| ENV
    CODE -->|3. Compila| DEPLOY
    ENV -->|4. Incluye| DEPLOY

    %% Build y Push
    DEPLOY -->|5. Construye imagen| ECR
    DEPLOY -->|6. Sube imagen| ECR
    DEPLOY -->|7. Sube configuración| S3
    DEPLOY -->|8. Sube variables| S3
    DEPLOY -->|9. Envía comando| SSM

    %% Despliegue en EC2
    SSM -->|10. Descarga configuración| S3
    SSM -->|11. Descarga variables| S3
    SSM -->|12. Login ECR| ECR
    SSM -->|13. Descarga imagen| ECR
    SSM -->|14. Inicia contenedor| DOCKER
    DOCKER -->|15. Ejecuta aplicación| CONTAINER

    %% Conexión a Base de Datos
    CONTAINER -->|16. Lee variables| S3
    CONTAINER -->|17. Conecta| RDS
    CONTAINER -->|18. Ejecuta migraciones| RDS

    %% Acceso Externo
    USER -->|19. HTTP Request| CONTAINER
    CONTAINER -->|20. Respuesta| USER
    DBA -->|21. Conexión directa| RDS

    %% Permisos
    EC2 -->|Usa permisos| IAM
    IAM -->|Permite acceso| S3
    IAM -->|Permite acceso| ECR
    IAM -->|Permite acceso| SSM

    %% Estilos
    classDef devStyle fill:#e1f5ff,stroke:#01579b,stroke-width:3px,color:#000
    classDef storageStyle fill:#fff4e1,stroke:#e65100,stroke-width:3px,color:#000
    classDef computeStyle fill:#e8f5e9,stroke:#2e7d32,stroke-width:3px,color:#000
    classDef dbStyle fill:#f3e5f5,stroke:#6a1b9a,stroke-width:3px,color:#000
    classDef mgmtStyle fill:#fce4ec,stroke:#c2185b,stroke-width:3px,color:#000
    classDef extStyle fill:#fff9c4,stroke:#f57f17,stroke-width:3px,color:#000

    class DEV,CODE,ENV,DEPLOY devStyle
    class S3,ECR storageStyle
    class EC2,DOCKER,CONTAINER computeStyle
    class RDS dbStyle
    class SSM,IAM mgmtStyle
    class USER,DBA extStyle
```

---

## 🎯 Flujo Paso a Paso (Versión Narrativa)

```mermaid
sequenceDiagram
    participant Dev as 👨‍💻 Desarrollador
    participant Code as 📝 Código
    participant Deploy as 🚀 deploy.sh
    participant ECR as 🐳 ECR
    participant S3 as 🪣 S3
    participant SSM as 📡 SSM
    participant EC2 as 🖥️ EC2
    participant Docker as 🐳 Docker
    participant Container as 📦 Contenedor
    participant RDS as 💾 RDS

    Note over Dev,Code: FASE 1: Preparación
    Dev->>Code: 1. Edita código
    Dev->>Code: 2. Crea env.production
    Dev->>Deploy: 3. Ejecuta deploy.sh

    Note over Deploy,ECR: FASE 2: Build y Push
    Deploy->>ECR: 4. Construye imagen Docker
    Deploy->>ECR: 5. Sube imagen a ECR
    Deploy->>S3: 6. Sube docker-compose.yml
    Deploy->>S3: 7. Sube env.production

    Note over Deploy,SSM: FASE 3: Comando de Despliegue
    Deploy->>SSM: 8. Envía comando de despliegue

    Note over SSM,EC2: FASE 4: Despliegue en EC2
    SSM->>S3: 9. Descarga docker-compose.yml
    SSM->>S3: 10. Descarga env.production
    SSM->>ECR: 11. Login y descarga imagen
    SSM->>Docker: 12. Inicia contenedor
    Docker->>Container: 13. Contenedor corriendo

    Note over Container,RDS: FASE 5: Conexión y Migraciones
    Container->>S3: 14. Lee env.production
    Container->>RDS: 15. Conecta a base de datos
    Container->>RDS: 16. Ejecuta migraciones SQL

    Note over Container: FASE 6: Aplicación Lista
    Container->>Container: 17. Servidor HTTP activo<br/>Puerto 8200
```

---

## 🏗️ Arquitectura del Sistema (Vista General)

```mermaid
graph TB
    subgraph LOCAL["🏠 DESARROLLO LOCAL"]
        DEV["👨‍💻 Desarrollador"]
        LAPTOP["💻 Laptop/PC"]
    end

    subgraph CLOUD["☁️ AWS CLOUD"]
        subgraph STORE["📦 ALMACENAMIENTO"]
            S3_BUCKET["🪣 S3<br/>Archivos de Configuración"]
            ECR_REPO["🐳 ECR<br/>Imágenes Docker"]
        end

        subgraph SERVER["🖥️ SERVIDOR"]
            EC2_SERVER["🖥️ EC2<br/>Ubuntu Server"]
            DOCKER_ENGINE["🐳 Docker"]
            APP_CONTAINER["📦 Aplicación<br/>Puerto 8200"]
        end

        subgraph DATA["💾 DATOS"]
            POSTGRES["🗄️ PostgreSQL<br/>Base de Datos"]
        end

        subgraph CONTROL["⚙️ CONTROL"]
            SSM_AGENT["📡 SSM<br/>Sistema de Control"]
            PERMISSIONS["🔑 Permisos IAM"]
        end
    end

    subgraph USERS["👥 USUARIOS"]
        END_USER["👤 Usuario Final"]
        ADMIN["👨‍💼 Administrador"]
    end

    %% Conexiones
    DEV -->|Desarrolla| LAPTOP
    LAPTOP -->|Sube código| ECR_REPO
    LAPTOP -->|Sube config| S3_BUCKET
    LAPTOP -->|Envía comando| SSM_AGENT

    SSM_AGENT -->|Descarga| S3_BUCKET
    SSM_AGENT -->|Descarga| ECR_REPO
    SSM_AGENT -->|Controla| EC2_SERVER

    EC2_SERVER -->|Ejecuta| DOCKER_ENGINE
    DOCKER_ENGINE -->|Corre| APP_CONTAINER
    APP_CONTAINER -->|Lee datos| POSTGRES
    APP_CONTAINER -->|Guarda datos| POSTGRES

    END_USER -->|Usa aplicación| APP_CONTAINER
    ADMIN -->|Administra| POSTGRES

    EC2_SERVER -->|Usa| PERMISSIONS
    PERMISSIONS -->|Permite acceso| S3_BUCKET
    PERMISSIONS -->|Permite acceso| ECR_REPO

    %% Estilos
    style LOCAL fill:#e3f2fd,stroke:#1976d2,stroke-width:2px
    style CLOUD fill:#f1f8e9,stroke:#558b2f,stroke-width:2px
    style STORE fill:#fff3e0,stroke:#f57c00,stroke-width:2px
    style SERVER fill:#e8f5e9,stroke:#388e3c,stroke-width:2px
    style DATA fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px
    style CONTROL fill:#fce4ec,stroke:#c2185b,stroke-width:2px
    style USERS fill:#fff9c4,stroke:#f9a825,stroke-width:2px
```

---

## 📋 Resumen Visual del Flujo

```mermaid
flowchart LR
    A["👨‍💻<br/>Desarrollador<br/>Edita código"] --> B["🔨<br/>Build<br/>Compila aplicación"]
    B --> C["📤<br/>Push<br/>Sube a ECR"]
    C --> D["📦<br/>Upload<br/>Sube config a S3"]
    D --> E["📡<br/>SSM<br/>Envía comando"]
    E --> F["⬇️<br/>Download<br/>Descarga en EC2"]
    F --> G["🐳<br/>Docker<br/>Inicia contenedor"]
    G --> H["💾<br/>RDS<br/>Conecta base datos"]
    H --> I["✅<br/>Listo<br/>Aplicación corriendo"]

    style A fill:#e1f5ff,stroke:#01579b,stroke-width:2px
    style B fill:#fff4e1,stroke:#e65100,stroke-width:2px
    style C fill:#fff4e1,stroke:#e65100,stroke-width:2px
    style D fill:#fff4e1,stroke:#e65100,stroke-width:2px
    style E fill:#fce4ec,stroke:#c2185b,stroke-width:2px
    style F fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px
    style G fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px
    style H fill:#f3e5f5,stroke:#6a1b9a,stroke-width:2px
    style I fill:#c8e6c9,stroke:#1b5e20,stroke-width:3px
```

---

## 🎨 Componentes del Sistema (Vista de Componentes)

```mermaid
graph TB
    subgraph "🌐 CAPA DE PRESENTACIÓN"
        USER["👤 Usuario Final"]
        ADMIN["👨‍💼 Administrador"]
    end

    subgraph "🖥️ CAPA DE APLICACIÓN"
        APP["📦 Aplicación Go<br/>ewood-backend<br/><br/>✅ Servidor HTTP<br/>✅ Endpoints API<br/>✅ Lógica de Negocio"]
    end

    subgraph "💾 CAPA DE DATOS"
        DB["🗄️ PostgreSQL RDS<br/><br/>📊 Tablas<br/>💾 Datos<br/>🔐 Seguridad"]
    end

    subgraph "☁️ CAPA DE INFRAESTRUCTURA"
        SERVER["🖥️ EC2 Instance<br/>Ubuntu Server"]
        STORAGE["📦 S3 + ECR<br/>Almacenamiento"]
        CONTROL["⚙️ SSM + IAM<br/>Control y Permisos"]
    end

    USER -->|HTTP Request| APP
    APP -->|HTTP Response| USER
    ADMIN -->|SQL Query| DB
    DB -->|SQL Result| ADMIN

    APP -->|Lee/Escribe| DB
    APP -->|Corre en| SERVER
    APP -->|Lee config de| STORAGE
    SERVER -->|Usa| CONTROL

    style USER fill:#fff9c4,stroke:#f9a825,stroke-width:2px
    style ADMIN fill:#fff9c4,stroke:#f9a825,stroke-width:2px
    style APP fill:#e8f5e9,stroke:#2e7d32,stroke-width:3px
    style DB fill:#f3e5f5,stroke:#6a1b9a,stroke-width:3px
    style SERVER fill:#e1f5ff,stroke:#01579b,stroke-width:2px
    style STORAGE fill:#fff4e1,stroke:#e65100,stroke-width:2px
    style CONTROL fill:#fce4ec,stroke:#c2185b,stroke-width:2px
```

---

## 📊 Dónde se Guarda Cada Cosa (Mapa Visual)

```mermaid
mindmap
  root((Sistema de Despliegue))
    Código Fuente
      GitHub
      Local
    Imagen Docker
      ECR
      Contiene código compilado
    Configuración
      S3
      docker-compose.yml
      env.production
    Aplicación
      EC2
      Contenedor Docker
      Puerto 8200
    Base de Datos
      RDS PostgreSQL
      Tablas
      Datos
    Control
      SSM
      IAM
      Permisos
```

---

## 🎯 Flujo Simplificado para No Técnicos

```mermaid
flowchart TD
    START([👨‍💻 Desarrollador<br/>tiene código listo]) --> STEP1[📝 Paso 1:<br/>Compila el código<br/>en una imagen Docker]
    STEP1 --> STEP2[📤 Paso 2:<br/>Sube la imagen<br/>a AWS ECR]
    STEP2 --> STEP3[📦 Paso 3:<br/>Sube archivos<br/>de configuración<br/>a AWS S3]
    STEP3 --> STEP4[📡 Paso 4:<br/>Envía comando<br/>al servidor EC2]
    STEP4 --> STEP5[⬇️ Paso 5:<br/>El servidor descarga<br/>todo lo necesario]
    STEP5 --> STEP6[🐳 Paso 6:<br/>Inicia el contenedor<br/>con la nueva versión]
    STEP6 --> STEP7[💾 Paso 7:<br/>Conecta a la base<br/>de datos]
    STEP7 --> STEP8[✅ Paso 8:<br/>Aplicación lista<br/>y funcionando]
    STEP8 --> END([🎉 Usuarios pueden<br/>usar la aplicación])

    style START fill:#e1f5ff,stroke:#01579b,stroke-width:3px
    style STEP1 fill:#fff4e1,stroke:#e65100,stroke-width:2px
    style STEP2 fill:#fff4e1,stroke:#e65100,stroke-width:2px
    style STEP3 fill:#fff4e1,stroke:#e65100,stroke-width:2px
    style STEP4 fill:#fce4ec,stroke:#c2185b,stroke-width:2px
    style STEP5 fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px
    style STEP6 fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px
    style STEP7 fill:#f3e5f5,stroke:#6a1b9a,stroke-width:2px
    style STEP8 fill:#c8e6c9,stroke:#1b5e20,stroke-width:3px
    style END fill:#c8e6c9,stroke:#1b5e20,stroke-width:3px
```

---

## 💡 Explicación Simple

**¿Qué pasa cuando desplegamos?**

1. **El desarrollador** tiene código nuevo listo
2. **Se compila** el código en una "caja" (imagen Docker)
3. **Se sube** esa "caja" a un almacén (ECR)
4. **Se suben** las instrucciones (docker-compose.yml) a otro almacén (S3)
5. **Se envía** un mensaje al servidor (SSM) diciendo "¡actualízate!"
6. **El servidor** descarga todo lo nuevo
7. **El servidor** detiene la versión vieja y inicia la nueva
8. **La aplicación** se conecta a la base de datos
9. **¡Listo!** Los usuarios pueden usar la nueva versión

**Todo esto pasa automáticamente en menos de 5 minutos** ⚡

