# ⚠️ CONFIGURACIÓN DE RDS PARA PRODUCCIÓN

## 🔒 Cambios Requeridos Antes de Producción

El RDS está actualmente configurado para **desarrollo/testing** con acceso público habilitado. **DEBES** hacer los siguientes cambios antes de pasar a producción:

---

## 📋 Checklist de Seguridad para Producción

### 1. Deshabilitar Acceso Público

**Archivo:** `terraform/main.tf`

**Cambiar:**
```hcl
allow_public_access = true  # ⚠️ DESARROLLO
```

**Por:**
```hcl
allow_public_access = false  # ✅ PRODUCCIÓN
```

**También en:** `terraform/modules/rds/variables.tf`
```hcl
default = false  # ⚠️ Cambiar de true a false
```

### 2. Habilitar Deletion Protection

**Archivo:** `terraform/modules/rds/main.tf`

**Cambiar:**
```hcl
deletion_protection = false  # ⚠️ DESARROLLO
```

**Por:**
```hcl
deletion_protection = true  # ✅ PRODUCCIÓN
```

### 3. Habilitar Final Snapshot

**Archivo:** `terraform/modules/rds/main.tf`

**Cambiar:**
```hcl
skip_final_snapshot = true  # ⚠️ DESARROLLO
```

**Por:**
```hcl
skip_final_snapshot = false
final_snapshot_identifier = "ewood-postgres-final-snapshot-${formatdate("YYYY-MM-DD-hhmm", timestamp())}"
```

### 4. Verificar Security Groups

Después de cambiar `allow_public_access = false`, la regla de ingress que permite `0.0.0.0/0` se eliminará automáticamente. Solo quedará la regla que permite conexiones desde el Security Group de EC2.

---

## 🚀 Pasos para Aplicar Cambios

1. **Editar los archivos** mencionados arriba
2. **Revisar el plan:**
   ```bash
   cd terraform
   terraform plan
   ```
3. **Aplicar los cambios:**
   ```bash
   terraform apply
   ```

---

## 🔍 Verificación Post-Cambio

Después de aplicar los cambios, verifica:

1. **RDS no es públicamente accesible:**
   ```bash
   aws rds describe-db-instances \
     --db-instance-identifier ewood-postgres \
     --query 'DBInstances[0].PubliclyAccessible' \
     --output text
   ```
   Debe retornar: `False`

2. **Security Group solo permite EC2:**
   ```bash
   aws ec2 describe-security-groups \
     --group-names rds-ewood-postgres \
     --query 'SecurityGroups[0].IpPermissions'
   ```
   No debe haber reglas con `0.0.0.0/0`

3. **Deletion Protection activo:**
   ```bash
   aws rds describe-db-instances \
     --db-instance-identifier ewood-postgres \
     --query 'DBInstances[0].DeletionProtection' \
     --output text
   ```
   Debe retornar: `True`

---

## 📝 Notas Importantes

- ⚠️ **Después de deshabilitar acceso público**, solo podrás conectarte al RDS desde:
  - La instancia EC2 (vía SSH/SSM)
  - Una VPN conectada a la VPC
  - Un bastion host en la VPC

- 🔐 **Para desarrollo local**, considera usar:
  - VPN a la VPC
  - Bastion host (jump server)
  - SSH tunnel desde EC2

- 📊 **Backups**: Los backups automáticos están configurados para 7 días. Considera aumentar en producción según tus necesidades.

---

## 🆘 Si Necesitas Acceso Temporal en Producción

Si necesitas acceso temporal para debugging (NO recomendado en producción):

1. **Habilitar temporalmente:**
   ```hcl
   allow_public_access = true
   ```

2. **Aplicar:**
   ```bash
   terraform apply
   ```

3. **Hacer lo que necesites**

4. **Deshabilitar inmediatamente:**
   ```hcl
   allow_public_access = false
   ```

5. **Aplicar de nuevo:**
   ```bash
   terraform apply
   ```

---

## ✅ Configuración Final Recomendada para Producción

```hcl
# terraform/main.tf
module "rds" {
  # ... otras configuraciones ...
  allow_public_access = false  # ✅ Solo acceso desde VPC
}

# terraform/modules/rds/main.tf
resource "aws_db_instance" "postgres" {
  # ... otras configuraciones ...
  publicly_accessible    = false
  deletion_protection     = true
  skip_final_snapshot    = false
  final_snapshot_identifier = "ewood-postgres-final-snapshot-${formatdate("YYYY-MM-DD-hhmm", timestamp())}"
}
```

