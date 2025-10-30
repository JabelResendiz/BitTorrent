# Ejemplo de Output con Logging Detallado

## 🎬 Escenario

- **Archivo**: vid.torrent (10 MB, 10 piezas de 1 MB cada una)
- **Peers**:
  - Seeder: `10.0.1.5:6881` (tiene todas las piezas)
  - Leecher1: `10.0.1.6:6881` (tiene piezas 0-4)
  - Leecher2: `10.0.1.7:6881` (recién se une, no tiene nada)

## 📺 Output Completo de Leecher2

```
Cliente BitTorrent iniciado
Conectando a tracker http://tracker:8000/announce
Tracker respondió: 2 peers disponibles

Conectando a peer 10.0.1.5:6881...
Handshake exitoso con peer 10.0.1.5:6881
Bitfield inicial recibido

Conectando a peer 10.0.1.6:6881...
Handshake exitoso con peer 10.0.1.6:6881
Bitfield inicial recibido

Peer te unchokeo. Buscando pieza a solicitar...
Descargando pieza 0 desde 2 peers en paralelo (Round-Robin)
  → Solicitando bloque 0 de pieza 0 a peer 10.0.1.5:6881
  → Solicitando bloque 1 de pieza 0 a peer 10.0.1.6:6881
  → Solicitando bloque 2 de pieza 0 a peer 10.0.1.5:6881
  → Solicitando bloque 3 de pieza 0 a peer 10.0.1.6:6881
  → Solicitando bloque 4 de pieza 0 a peer 10.0.1.5:6881
  → Solicitando bloque 5 de pieza 0 a peer 10.0.1.6:6881
  → Solicitando bloque 6 de pieza 0 a peer 10.0.1.5:6881
  → Solicitando bloque 7 de pieza 0 a peer 10.0.1.6:6881
  → Solicitando bloque 8 de pieza 0 a peer 10.0.1.5:6881
  → Solicitando bloque 9 de pieza 0 a peer 10.0.1.6:6881
  → Solicitando bloque 10 de pieza 0 a peer 10.0.1.5:6881
  → Solicitando bloque 11 de pieza 0 a peer 10.0.1.6:6881
  → Solicitando bloque 12 de pieza 0 a peer 10.0.1.5:6881
  → Solicitando bloque 13 de pieza 0 a peer 10.0.1.6:6881
  → Solicitando bloque 14 de pieza 0 a peer 10.0.1.5:6881
  → Solicitando bloque 15 de pieza 0 a peer 10.0.1.6:6881
  ... (continúa hasta bloque 63)

✓ Recibido bloque 0 de pieza 0 desde peer 10.0.1.5:6881 (offset 0, tamaño 16384 bytes)
✓ Recibido bloque 1 de pieza 0 desde peer 10.0.1.6:6881 (offset 16384, tamaño 16384 bytes)
✓ Recibido bloque 2 de pieza 0 desde peer 10.0.1.5:6881 (offset 32768, tamaño 16384 bytes)
✓ Recibido bloque 3 de pieza 0 desde peer 10.0.1.6:6881 (offset 49152, tamaño 16384 bytes)
✓ Recibido bloque 4 de pieza 0 desde peer 10.0.1.5:6881 (offset 65536, tamaño 16384 bytes)
✓ Recibido bloque 5 de pieza 0 desde peer 10.0.1.6:6881 (offset 81920, tamaño 16384 bytes)
✓ Recibido bloque 7 de pieza 0 desde peer 10.0.1.6:6881 (offset 114688, tamaño 16384 bytes)
✓ Recibido bloque 6 de pieza 0 desde peer 10.0.1.5:6881 (offset 98304, tamaño 16384 bytes)
✓ Recibido bloque 8 de pieza 0 desde peer 10.0.1.5:6881 (offset 131072, tamaño 16384 bytes)
✓ Recibido bloque 9 de pieza 0 desde peer 10.0.1.6:6881 (offset 147456, tamaño 16384 bytes)
... (continúa recibiendo bloques en paralelo)
✓ Recibido bloque 63 de pieza 0 desde peer 10.0.1.6:6881 (offset 1032192, tamaño 16384 bytes)

═══════════════════════════════════════════════
✓ Pieza 0 completada (Round-Robin)
═══════════════════════════════════════════════
Distribución de bloques por peer:
  • Peer 10.0.1.5:6881: 32 bloques
  • Peer 10.0.1.6:6881: 32 bloques
Total: 64 bloques
═══════════════════════════════════════════════

Broadcast HAVE a 2 peers. Pieza 0

Peer te unchokeo. Buscando pieza a solicitar...
Descargando pieza 1 desde 2 peers en paralelo (Round-Robin)
  → Solicitando bloque 0 de pieza 1 a peer 10.0.1.5:6881
  → Solicitando bloque 1 de pieza 1 a peer 10.0.1.6:6881
  → Solicitando bloque 2 de pieza 1 a peer 10.0.1.5:6881
  ... (continúa con pieza 1)

✓ Recibido bloque 0 de pieza 1 desde peer 10.0.1.5:6881 (offset 0, tamaño 16384 bytes)
✓ Recibido bloque 1 de pieza 1 desde peer 10.0.1.6:6881 (offset 16384, tamaño 16384 bytes)
... (continúa recibiendo)

═══════════════════════════════════════════════
✓ Pieza 1 completada (Round-Robin)
═══════════════════════════════════════════════
Distribución de bloques por peer:
  • Peer 10.0.1.5:6881: 32 bloques
  • Peer 10.0.1.6:6881: 32 bloques
Total: 64 bloques
═══════════════════════════════════════════════

Broadcast HAVE a 2 peers. Pieza 1

... (continúa con las demás piezas)

Descarga completada, no hay más piezas
Archivo vid.mp4 descargado exitosamente
```

## 🔍 Análisis del Output

### 1. Paralelismo Real

Observa que los bloques NO llegan en orden:
```
✓ Recibido bloque 5 ...
✓ Recibido bloque 7 ... ← Llegó antes que bloque 6
✓ Recibido bloque 6 ...
```

**Conclusión**: Los peers responden independientemente, confirmando descarga paralela.

### 2. Distribución Perfecta

En el resumen de pieza 0:
```
Peer 10.0.1.5:6881: 32 bloques (50%)
Peer 10.0.1.6:6881: 32 bloques (50%)
```

**Conclusión**: Round-Robin logra balance perfecto cuando hay 2 peers y 64 bloques (par).

### 3. Distribución con 3 Peers

Si hubiera 3 peers conectados:
```
Peer 10.0.1.5:6881: 22 bloques (34.4%)
Peer 10.0.1.6:6881: 21 bloques (32.8%)
Peer 10.0.1.7:6881: 21 bloques (32.8%)
Total: 64 bloques
```

**Conclusión**: Un peer recibe 1 bloque extra (64 no es divisible por 3).

## 📊 Comparación Visual: Antes vs Después

### ❌ Output Anterior (Sin Logging Detallado)

```
Peer te unchokeo. Buscando pieza a solicitar...
Recibido block de pieza 0, offset 0, tamaño 16384 bytes
Recibido block de pieza 0, offset 16384, tamaño 16384 bytes
Recibido block de pieza 0, offset 32768, tamaño 16384 bytes
... (sin info de origen)
Pieza 0 completa (Round-Robin)
```

**Problemas**:
- ❌ No se sabe qué peer envió cada bloque
- ❌ No se puede verificar distribución Round-Robin
- ❌ Difícil debuggear problemas de peers

### ✅ Output Actual (Con Logging Detallado)

```
Descargando pieza 0 desde 2 peers en paralelo (Round-Robin)
  → Solicitando bloque 0 de pieza 0 a peer 10.0.1.5:6881
  → Solicitando bloque 1 de pieza 0 a peer 10.0.1.6:6881
  
✓ Recibido bloque 0 de pieza 0 desde peer 10.0.1.5:6881 (offset 0, tamaño 16384 bytes)
✓ Recibido bloque 1 de pieza 0 desde peer 10.0.1.6:6881 (offset 16384, tamaño 16384 bytes)

═══════════════════════════════════════════════
Distribución de bloques por peer:
  • Peer 10.0.1.5:6881: 32 bloques
  • Peer 10.0.1.6:6881: 32 bloques
═══════════════════════════════════════════════
```

**Ventajas**:
- ✅ Clara visibilidad de qué peer maneja cada bloque
- ✅ Fácil verificar algoritmo Round-Robin
- ✅ Estadísticas útiles para optimización
- ✅ Debugging más simple

## 🎯 Casos de Uso del Logging

### 1. Verificar Round-Robin
Comparar secuencia de solicitudes con peers asignados:
```
Bloque 0 → Peer1 ✓
Bloque 1 → Peer2 ✓
Bloque 2 → Peer1 ✓
```

### 2. Detectar Peers Lentos
```
Peer 10.0.1.5:6881: 32 bloques (recibidos en 500ms)
Peer 10.0.1.6:6881: 32 bloques (recibidos en 2000ms) ← Lento
```

### 3. Identificar Peers que Fallan
```
→ Solicitando bloque 5 a peer 10.0.1.7:6881
✗ (Sin respuesta después de 30s)
```

### 4. Medir Throughput Real
```
Pieza 0: 1 MB / 220ms = 4.54 MB/s (2 peers)
Pieza 1: 1 MB / 150ms = 6.67 MB/s (3 peers)
```

## 🐛 Debugging con los Logs

### Problema: Bloques no llegan

**Síntoma**:
```
→ Solicitando bloque 10 a peer 10.0.1.5:6881
→ Solicitando bloque 11 a peer 10.0.1.6:6881
✓ Recibido bloque 11 desde peer 10.0.1.6:6881
(bloque 10 nunca llega)
```

**Diagnóstico**: Peer 10.0.1.5 tiene problema de conexión.

### Problema: Distribución desbalanceada

**Síntoma**:
```
Peer 10.0.1.5:6881: 64 bloques
Peer 10.0.1.6:6881: 0 bloques
```

**Diagnóstico**: Peer2 no está recibiendo REQUEST (bug en Round-Robin).

### Problema: Duplicados

**Síntoma**:
```
✓ Recibido bloque 5 desde peer 10.0.1.5:6881
✓ Recibido bloque 5 desde peer 10.0.1.6:6881 ← Duplicado
```

**Diagnóstico**: Se enviaron 2 REQUEST para el mismo bloque (bug en tracking).

## 📈 Métricas Calculables

Con estos logs puedes calcular:

1. **Latencia promedio por peer**:
   - Tiempo entre REQUEST y PIECE

2. **Throughput por peer**:
   - Bytes recibidos / tiempo total

3. **Tasa de pérdida**:
   - REQUEST sin PIECE correspondiente

4. **Eficiencia Round-Robin**:
   - Desviación estándar de bloques por peer

5. **Paralelismo real**:
   - Bloques recibidos fuera de orden / total bloques

---

**Última actualización**: 26 de Octubre, 2025  
**Versión**: v2.1.0  
**Mejora**: Visibilidad completa del flujo Round-Robin
