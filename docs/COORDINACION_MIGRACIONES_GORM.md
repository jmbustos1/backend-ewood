# 🔄 Coordinación entre Migraciones SQL y Modelos GORM

## 📋 Resumen

Este documento explica cómo coordinar las **migraciones SQL** (golang-migrate) con los **modelos GORM** para mantener la consistencia entre el esquema de la base de datos y el código.

---

## 🎯 Estrategia: Migraciones SQL Primero

### ✅ Flujo Recomendado

```
1. Diseñar esquema → 2. Crear migración SQL → 3. Crear modelo GORM → 4. Usar en código
```

**Regla de oro**: Las migraciones SQL son la **fuente de verdad** del esquema. Los modelos GORM deben reflejar exactamente lo que está en las migraciones.

---

## 📝 Proceso de Trabajo

### Paso 1: Crear Migración SQL

Primero, creas la migración SQL con el esquema:

```sql
-- migrations/000002_create_scooters_schema.up.sql
CREATE TABLE scooters (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'available',
    ...
);
```

### Paso 2: Crear Modelo GORM

Luego, creas el modelo GORM que refleja la tabla:

```go
// internal/models/scooter.go
type Scooter struct {
    ID     uint   `gorm:"primaryKey"`
    Code   string `gorm:"uniqueIndex;not null;size:50"`
    Status string `gorm:"type:varchar(20);not null;default:'available'"`
    ...
}
```

### Paso 3: Sincronizar Tags GORM

Los tags GORM deben coincidir con la migración SQL:

| SQL | GORM Tag |
|-----|----------|
| `SERIAL PRIMARY KEY` | `gorm:"primaryKey"` |
| `VARCHAR(50) NOT NULL UNIQUE` | `gorm:"uniqueIndex;not null;size:50"` |
| `VARCHAR(20) NOT NULL DEFAULT 'available'` | `gorm:"type:varchar(20);not null;default:'available'"` |
| `DECIMAL(10,8)` | `gorm:"type:decimal(10,8)"` |
| `TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP` | `gorm:"not null;default:CURRENT_TIMESTAMP"` |
| `FOREIGN KEY` | `gorm:"foreignKey:FieldName"` |

---

## ⚠️ NO Usar AutoMigrate en Producción

### ❌ No hagas esto:

```go
// NO usar AutoMigrate para crear tablas en producción
db.AutoMigrate(&Scooter{}, &Trip{})
```

**Razones:**
- AutoMigrate puede hacer cambios inesperados
- No hay control de versiones
- No hay rollback
- Puede perder datos

### ✅ Haz esto:

```go
// Solo usar AutoMigrate para desarrollo local/testing
// En producción, usa migraciones SQL
if os.Getenv("ENV") == "development" {
    db.AutoMigrate(&Scooter{}, &Trip{})
}
```

---

## 🔄 Flujo Completo de Desarrollo

### 1. Desarrollo de Nueva Feature

```bash
# 1. Crear migración SQL
touch migrations/000003_add_new_field.up.sql
touch migrations/000003_add_new_field.down.sql

# 2. Escribir SQL
# migrations/000003_add_new_field.up.sql
ALTER TABLE scooters ADD COLUMN new_field VARCHAR(100);

# 3. Actualizar modelo GORM
type Scooter struct {
    ...
    NewField string `gorm:"size:100" json:"new_field"`
}

# 4. Probar localmente
make migrate-up
go run ./cmd/server

# 5. Commit y push
git add migrations/ internal/models/
git commit -m "Add new_field to scooters"
```

### 2. Despliegue a Producción

```bash
# 1. Desplegar código (incluye migraciones)
./deploy.sh

# 2. Las migraciones se ejecutan automáticamente
# (ya está configurado en deploy.sh)
```

---

## 📊 Checklist de Sincronización

Antes de hacer commit, verifica:

- [ ] Migración SQL creada (`.up.sql` y `.down.sql`)
- [ ] Modelo GORM actualizado con los mismos campos
- [ ] Tags GORM coinciden con tipos SQL
- [ ] Índices definidos en ambos (SQL y GORM)
- [ ] Foreign keys definidas en ambos
- [ ] Constraints (CHECK, NOT NULL, etc.) en ambos
- [ ] Valores por defecto coinciden
- [ ] Migración probada localmente

---

## 🛠️ Herramientas Útiles

### Verificar Estructura de Tabla

```sql
-- Ver estructura de tabla en PostgreSQL
\d scooters

-- Ver índices
\di scooters
```

### Comparar Modelo GORM con Tabla Real

```go
// Función de utilidad para verificar
func VerifyTableStructure(db *gorm.DB, model interface{}) error {
    return db.AutoMigrate(model) // Solo verifica, no modifica si ya existe
}
```

---

## 🎯 Buenas Prácticas

### 1. Nombres Consistentes

- **Tablas**: `snake_case` (ej: `scooter_locations`)
- **Modelos Go**: `PascalCase` (ej: `ScooterLocation`)
- **Campos**: `snake_case` en DB, `PascalCase` en Go

### 2. Tipos de Datos

| SQL | Go | GORM Tag |
|-----|----|----|
| `SERIAL` | `uint` | `gorm:"primaryKey"` |
| `VARCHAR(n)` | `string` | `gorm:"size:n"` |
| `DECIMAL(p,s)` | `*float64` | `gorm:"type:decimal(p,s)"` |
| `TIMESTAMP` | `time.Time` | `gorm:"type:timestamp"` |
| `BOOLEAN` | `bool` | `gorm:"type:boolean"` |
| `INTEGER` | `int` | `gorm:"type:integer"` |

### 3. Relaciones

```go
// One-to-Many
type Scooter struct {
    Trips []Trip `gorm:"foreignKey:ScooterID"`
}

// Many-to-One
type Trip struct {
    Scooter Scooter `gorm:"foreignKey:ScooterID"`
}

// Many-to-One (nullable)
type ScooterLocation struct {
    Trip *Trip `gorm:"foreignKey:TripID"`
}
```

---

## 🚨 Errores Comunes

### Error 1: Desincronización

**Problema**: Modelo GORM no coincide con tabla SQL

```go
// Modelo tiene campo que no existe en DB
type Scooter struct {
    NonExistentField string // ❌ No existe en migración SQL
}
```

**Solución**: Siempre actualizar migración SQL primero, luego modelo GORM

### Error 2: AutoMigrate en Producción

**Problema**: Usar AutoMigrate para crear tablas en producción

```go
db.AutoMigrate(&Scooter{}) // ❌ No en producción
```

**Solución**: Usar solo migraciones SQL en producción

### Error 3: Tipos Incompatibles

**Problema**: Tipo Go no coincide con tipo SQL

```sql
-- SQL
battery_level INTEGER
```

```go
// Go - ❌ Incorrecto
BatteryLevel string

// Go - ✅ Correcto
BatteryLevel int
```

---

## 📚 Ejemplo Completo

### Migración SQL

```sql
-- migrations/000002_create_scooters.up.sql
CREATE TABLE scooters (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    battery_level INTEGER NOT NULL DEFAULT 100,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_scooters_code ON scooters(code);
```

### Modelo GORM

```go
// internal/models/scooter.go
type Scooter struct {
    ID          uint      `gorm:"primaryKey" json:"id"`
    Code        string    `gorm:"uniqueIndex;not null;size:50" json:"code"`
    BatteryLevel int      `gorm:"not null;default:100" json:"battery_level"`
    CreatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (Scooter) TableName() string {
    return "scooters"
}
```

### Uso en Código

```go
// internal/service/scooter_service.go
func (s *ScooterService) CreateScooter(code string) (*models.Scooter, error) {
    scooter := &models.Scooter{
        Code: code,
        BatteryLevel: 100,
    }
    
    if err := s.db.Create(scooter).Error; err != nil {
        return nil, err
    }
    
    return scooter, nil
}
```

---

## ✅ Resumen

1. **Migraciones SQL** = Fuente de verdad del esquema
2. **Modelos GORM** = Reflejan las migraciones SQL
3. **NO usar AutoMigrate** en producción
4. **Siempre probar** migraciones localmente antes de desplegar
5. **Mantener sincronizados** SQL y GORM

---

## 🔗 Referencias

- [GORM Documentation](https://gorm.io/docs/)
- [golang-migrate Documentation](https://github.com/golang-migrate/migrate)
- [PostgreSQL Data Types](https://www.postgresql.org/docs/current/datatype.html)

