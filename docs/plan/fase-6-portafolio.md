# Fase 6 — Portafolio

Convertir Cuadra en una pieza de portafolio con números reales. Se puede trabajar durante el piloto; T53 es opcional.

> El README (T52) conviene irlo actualizando desde antes; esta tarea es la versión final y pulida.

Cada tarea: `/tarea TXX`. Las tareas van en orden de ejecución.

---

### [ ] T51 · Pruebas de carga (k6)
**Alcance:** `loadtest/` con escenarios de k6: búsqueda de productos en el POS, registro de venta y concurrencia sobre el mismo producto. Datos realistas (varios tenants, miles de productos). Resultados exportados a Prometheus y visibles en Grafana. Documentar en `docs/rendimiento.md`: hardware, escenario, throughput y p95/p99, y los cuellos de botella encontrados y corregidos.
**Aceptación:** un informe con números reales y al menos una mejora justificada con un antes y un después.

### [ ] T52 · README y documentación de arquitectura (portafolio)
**Alcance:** README con propuesta de valor, capturas de la UI, diagramas **C4** (contexto, contenedores y componentes) en Mermaid o Structurizr, el stack y por qué, decisiones destacadas (enlazando a `docs/decisiones.md`), capturas de dashboards y trazas, resultados de carga, y cómo levantarlo en local con un solo comando. Badges de CI y cobertura.
**Aceptación:** alguien sin contexto entiende qué es, cómo está construido y por qué, en 5 minutos.

### [ ] T53 · Laboratorio de Kubernetes (opcional, aprendizaje)
**Alcance:** **no** es para producción (producción corre en Container Apps; ver decisiones). Chart de Helm en `deploy/helm/cuadra/`: API y worker como Deployments, migraciones como Job previo (hook de Helm), ConfigMap y Secret, probes `/healthz` y `/readyz`, requests y limits, HPA por CPU y PodDisruptionBudget. Despliegue en un clúster local con **k3d** o **kind**, junto con el stack de observabilidad. CI: `helm lint` y validación de manifiestos con `kubeconform`. Documentar en `docs/kubernetes.md` cómo levantarlo y qué se aprendió (incluida la comparación honesta con Container Apps).
**Aceptación:** `make k8s-up` levanta Cuadra en un clúster local, escala con carga de k6 y las trazas siguen llegando a Grafana.
