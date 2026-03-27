# hawkings

CLI ligero para Hawkings pensado para contexto local, navegacion de la jerarquia de contenidos y consumo por agentes.

## Principios implementados

- Configuracion determinista con `hawkings.toml` buscado hacia arriba desde el directorio actual y `~/.hawkings.toml`.
- Salida `json` estable y `table` para uso interactivo.
- Resolucion explicita de perfil, entorno y `x-api-key`.
- Navegacion compacta de `program -> course -> section -> module -> content`.
- Comando `describe` para introspeccion del dominio y del propio CLI.
- Recuperacion explicita de contenido grande con truncado por defecto.

## Resolución de configuración

Orden de lectura:

1. Flags.
2. Variables de entorno `HAWKINGS_CLI_*`.
3. `hawkings.toml` mas cercano al directorio actual, buscado hacia arriba.
4. `~/.hawkings.toml`.

El fichero local es ideal para fijar solo el perfil:

```toml
profile = "dev"
```

El fichero del home contiene los perfiles y secretos:

```toml
version = 1
profile = "dev"

[profiles.dev]
environment = "dev"
x_api_key = "hk-..."
timeout = "30s"
```

## Comandos

```bash
go run ./cmd/hawkings config show
go run ./cmd/hawkings auth whoami
go run ./cmd/hawkings platform list
go run ./cmd/hawkings language list
go run ./cmd/hawkings faculty list --limit 10
go run ./cmd/hawkings template list
go run ./cmd/hawkings space list --limit 10
go run ./cmd/hawkings space get 48 --with coursePrograms
go run ./cmd/hawkings space create --json '{"name":"CLI Test Space","description":"Space creado desde hawkings","color":"#0EA5E9","personal":false,"enabled":true}' --dry-run
go run ./cmd/hawkings space update 48 --json '{"description":"Space actualizado desde hawkings"}' --dry-run
go run ./cmd/hawkings space programs 48
go run ./cmd/hawkings space assign-programs 48 --selected 330,331 --dry-run
go run ./cmd/hawkings space users 48
go run ./cmd/hawkings space assign-users 48 --selected 10 --dry-run
go run ./cmd/hawkings space delete 53 --dry-run
go run ./cmd/hawkings program create --json '{"name":"test2","language_id":2,"enabled":true,"status":"created","course_program_template_id":1,"metadata":{"hours":80,"hours_content_percentage":40}}' --space 48 --dry-run
go run ./cmd/hawkings program create --json-file ./examples/program-create.syllabus-processed.sample.json --syllabus-file ./examples/syllabus.sample.json --space 48 --dry-run
go run ./cmd/hawkings program update 330 --json '{"context":"contenido","metadata":{"code":"00xx22","description":"descripcion"}}' --dry-run
go run ./cmd/hawkings program delete 330 --dry-run
go run ./cmd/hawkings program set-spaces 330 --space 48 --dry-run
go run ./cmd/hawkings program set-courses 330 --course 2241,2242 --dry-run
go run ./cmd/hawkings program add-course 330 --course 2241 --dry-run
go run ./cmd/hawkings program remove-course 330 --course 2241 --dry-run
go run ./cmd/hawkings program generate-syllabus 330 --dry-run
go run ./cmd/hawkings program list --limit 10
go run ./cmd/hawkings program list --space-id 48 --limit 20
go run ./cmd/hawkings program get 329 --output table
go run ./cmd/hawkings program tree 316 --output table
go run ./cmd/hawkings program syllabus 327
go run ./cmd/hawkings program courses 316 --output table
go run ./cmd/hawkings program status-matrix --output table
go run ./cmd/hawkings course create --program 413 --json-file ./examples/course.sample.json --dry-run
go run ./cmd/hawkings course get 2241 --output table
go run ./cmd/hawkings course sections 2241 --output table
go run ./cmd/hawkings course modules 2241 --output table
go run ./cmd/hawkings section generate-content 987 --dry-run
go run ./cmd/hawkings section generate-activities 987 --dry-run
go run ./cmd/hawkings module create --course-id 2241 --type markdown --name "Nuevo modulo" --dry-run
go run ./cmd/hawkings module create --section-id 2156 --type markdown --name "Nuevo modulo de seccion"
go run ./cmd/hawkings module get 13721 --output table
go run ./cmd/hawkings module content 13721 --output table
go run ./cmd/hawkings module set-content 13721 --file ./contenido.md --dry-run
go run ./cmd/hawkings module set-content 13721 --content-file ./contenido.md --dry-run
go run ./cmd/hawkings module set-content 13721 --content '# Titulo\n\nTexto manual'
go run ./cmd/hawkings module update 13721 --json '{"status":"processed"}' --dry-run
go run ./cmd/hawkings module patch 13721 --json '{"status":"processed"}' --dry-run
go run ./cmd/hawkings module generate-content 13721 --dry-run
go run ./cmd/hawkings module approve 13721 --approved=false --dry-run
go run ./cmd/hawkings content approve 13721 --approved=true --dry-run
go run ./cmd/hawkings module content 13721 --raw --max-chars 1200
go run ./cmd/hawkings describe
go run ./cmd/hawkings describe hierarchy
go run ./cmd/hawkings describe entity module
go run ./cmd/hawkings describe command "module content"
```

## Jerarquia navegable

El CLI expone esta jerarquia:

```text
program -> course -> section -> module -> content
```

Puntos importantes:

- `program status` orienta, pero no garantiza que existan `courses`.
- `space` es una entidad transversal para organizar programas; puedes gestionarla desde el propio CLI.
- `language list`, `faculty list` y `template list` sirven para descubrir IDs de referencia sin salir del CLI.
- `space delete` solo funciona cuando el espacio no tiene programas asociados.
- `program create` acepta payload JSON y puede asignar spaces en la misma operacion.
- `program create` tambien puede inyectar `syllabus` desde `--syllabus-file`.
- `program update` envia directamente `PATCH /only`; el merge parcial de `metadata` lo resuelve ahora el backend.
- `program delete` no borra courses compartidos; el backend solo elimina los ligados exclusivamente a ese program.
- `program set-courses`, `program add-course` y `program remove-course` usan el endpoint dedicado `/course-program/{id}/course`.
- `program list --space-id` usa la membresia real del space y aplica search/status/paginacion en cliente.
- `program tree` navega estructura sin cargar `file.contents`.
- `course create` usa `/course/bulk` para crear el arbol y, si le pasas `--program`, relaciona despues el curso con `POST /course-program/{id}/course`.
- `section generate-content` y `section generate-activities` cubren la operativa bulk de `/module/results`.
- `module get` lista los `course_contents` de forma ligera.
- `module content` lee `course_contents[].file.contents` con truncado por defecto a `1000` caracteres.
- `module create` crea módulos de curso o de sección y calcula `order` automáticamente si no lo indicas.
- `module set-content` permite crear o actualizar el `course-content` manualmente a partir de `--file`, `--content-file` o `--content`, sin usar la generacion del modulo.
- `module update` es el alias natural de `module patch`; ambos llaman a `PATCH /course-module/{id}/only`.
- `module patch`, `module generate-content` y `module approve` cubren la operativa base de edición y generación a nivel de módulo.
- `content approve` es el alias semántico de “aprobar contenido” y usa el mismo `approved_at` del módulo que usa front.
- `describe` explica entidades, relaciones, expansiones `with[]` y comandos asociados.

## Creacion de courses via bulk

Para crear o actualizar un curso completo con secciones y modulos:

```bash
go run ./cmd/hawkings course create --program 413 --json-file ./examples/course.sample.json --dry-run
go run ./cmd/hawkings course create --program 413 --json '{"name":"Curso de olivos","language_id":2,"course_sections":[{"name":"Bloque 1","order":1,"course_modules":[{"name":"Introduccion","type":"markdown","order":1,"empty":true}]}]}'
```

Notas:

- `course_sections` es obligatorio en `/course/bulk`.
- Si un `module` es `markdown`, debe llevar `course_contents`, salvo que uses `empty=true`.
- Si usas `--program`, el CLI hace una segunda llamada con `add` a `/course-program/{id}/course`.
- El endpoint sincroniza el arbol enviado: si omites sections o modules existentes, el backend puede eliminarlos.
- El backend puede devolver `200 OK` con errores parciales dentro del JSON; `hawkings course create` ahora inspecciona esa respuesta y falla si detecta alguno.

## Escritura manual de contenido

Cuando un `module` esta en `empty`, puedes cargarle markdown manual sin pasar por `module generate-content`:

```bash
go run ./cmd/hawkings module set-content 22598 --file ./contenido.md
go run ./cmd/hawkings module set-content 22598 --content-file ./contenido.md
go run ./cmd/hawkings module set-content 22598 --content '# Introduccion\n\nTexto escrito a mano'
```

Comportamiento:

- Si el module no tiene `course_contents`, crea uno nuevo con `POST /course-content`.
- Si ya existe, actualiza el primero o el indicado con `--content-id` via `PATCH /course-content/{id}`.
- Despues hace `PATCH /course-module/{id}/only` con `status=processed`, salvo que cambies `--module-status`.
- Este flujo evita el endpoint `POST /course-module/{id}/course-content/generate`.
- El backend puede seguir calculando `summary` y metadatos derivados del contenido.

## Creacion de modules

Para crear un module nuevo sin construir JSON a mano:

```bash
go run ./cmd/hawkings module create --course-id 2323 --type markdown --name "Marco normativo del olivar"
go run ./cmd/hawkings module create --section-id 2156 --type markdown --name "Biologia avanzada del olivo"
```

Notas:

- Usa `--course-id` para modules de primer nivel del course.
- Usa `--section-id` para modules dentro de una section; el CLI resuelve `course_id` automaticamente.
- Si no pasas `--order`, el CLI calcula `max(order)+1` en ese ambito.
- Si pasas un `--order` ya ocupado, el backend desplaza los modulos siguientes.
- Para meter contenido manual justo despues, encadena `module set-content <module-id> --file ...`.

## Build

```bash
go build -o bin/hawkings ./cmd/hawkings
```

Cross-compile:

```bash
GOOS=linux GOARCH=amd64 go build -o dist/hawkings-linux-amd64 ./cmd/hawkings
GOOS=darwin GOARCH=arm64 go build -o dist/hawkings-darwin-arm64 ./cmd/hawkings
GOOS=windows GOARCH=amd64 go build -o dist/hawkings-windows-amd64.exe ./cmd/hawkings
```
