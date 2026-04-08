# hawkings

CLI para Hawkings.

Sirve para trabajar con:
- `program`
- `course`
- `scorm`
- `section`
- `module`
- `content`
- `space`

El binario funciona con un `hawkings.toml` local. No hace falta escribir nada en `~/.hawkings.toml` si no quieres.

## Instalacion

### Opcion 1: usar un binario ya compilado

Pon el binario en alguna carpeta de tu `PATH`.

macOS o Linux:

```bash
chmod +x ./hawkings
mkdir -p "$HOME/.local/bin"
mv ./hawkings "$HOME/.local/bin/hawkings"
```

Y añade esto a tu shell si hace falta:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Alternativa en macOS:

```bash
sudo mv ./hawkings /usr/local/bin/hawkings
```

### Opcion 2: compilar desde el repo

```bash
git clone <REPO_URL>
cd hawkings/hawkings-cli
go build -o bin/hawkings ./cmd/hawkings
```

Luego usa `bin/hawkings` o muvelo a tu `PATH`.

## Configuracion

Orden de lectura:

1. Flags
2. Variables de entorno `HAWKINGS_CLI_*`
3. `hawkings.toml` mas cercano al directorio actual
4. `~/.hawkings.toml`

Lo normal para compartirlo con otra persona es:

1. copiar [hawkings.toml.example](./hawkings.toml.example) a `hawkings.toml`
2. rellenar `x_api_key`
3. ejecutar el binario

Ejemplo:

```bash
cp hawkings.toml.example hawkings.toml
```

## hawkings.toml minimo

```toml
version = 1
profile = "dev"

[profiles.dev]
environment = "dev"
base_url = "https://dev-data-api.hawkings.education/v1"
x_api_key = ""
timeout = "30s"
```

Tambien puedes separar `api_key` y `platform_uuid`:

```toml
version = 1
profile = "dev"

[profiles.dev]
environment = "dev"
base_url = "https://dev-data-api.hawkings.education/v1"
api_key = ""
platform_uuid = ""
timeout = "30s"
```

## Primer uso

```bash
hawkings config show
hawkings auth whoami
hawkings program list --limit 10
hawkings course list --search "internet seguro"
```

## Paginacion

Los comandos `list` paginados devuelven por defecto una sola pagina.

En JSON siempre veras:
- `data`
- `page`
- `pages`
- `total`

Si quieres todos los resultados en una sola salida, usa `--all`.

```bash
hawkings program list --output json
hawkings program list --all --output json
hawkings course list --all --search "internet seguro"
hawkings space list --all
hawkings faculty list --all
```

## Comandos utiles

Buscar programas por texto:

```bash
hawkings program list --search "olivos"
```

Listar programas con prioridad custom por status y luego nombre:

```bash
hawkings program list \
  --order-column 'status;name' \
  --order-mode 'completed,processed,courses-created;ASC'
```

Buscar cursos por texto:

```bash
hawkings course list --search "internet seguro"
```

Ver la estructura de un programa:

```bash
hawkings program tree 5544
```

Ver un curso:

```bash
hawkings course get 35572
```

Ver modulos de un curso:

```bash
hawkings course modules 35572
```

Leer el contenido de un modulo:

```bash
hawkings module content 22598
hawkings module content 22598 --full --raw
```

Crear un curso completo y asociarlo a un programa:

```bash
hawkings course create --program 5544 --json-file ./course.json
```

Crear un recurso SCORM sin enviar `user_id` ni `language_id` aunque vengan en el JSON:

```bash
hawkings scorm create --json-file ./scorm.json
```

Crear los courses de un programa que ya tiene syllabus:

```bash
hawkings program create-courses 410
hawkings program create-courses 410 --force
hawkings --timeout 300s program create-courses 410
```

Escribir contenido manual en un modulo:

```bash
hawkings module set-content 22598 --content-file ./tema1.md
```

## Descubrir el CLI

```bash
hawkings describe
hawkings describe hierarchy
hawkings describe entity module
hawkings describe command "course create"
```

## Notas

- `program list --search` y `course list --search` hacen busqueda libre por texto.
- `program list`, `course list`, `space list` y `faculty list` soportan `--all` para recorrer todas las paginas de forma explicita.
- `program create-courses` llama a `/course-program/{id}/syllabus/course` y usa el syllabus ya guardado en el programa.
- `program create-courses` puede tardar minutos. Si el cliente hace timeout, verifica primero con `program get` o `program courses` antes de reintentar.
- `program list` acepta `--order-column` y `--order-mode`; por ejemplo `status;name` con `completed,processed,courses-created;ASC`.
- Para reutilizar cursos existentes: `hawkings course list --all` para descubrir IDs y luego `hawkings program add-course <program-id> --course <course-id>`.
- `course create --program` crea el curso via `/course/bulk` y luego lo relaciona con `POST /course-program/{id}/course`.
- `scorm create` sanea el payload y no envia `user_id` ni `language_id`.
- `module content` trunca por defecto para no saturar contexto. Usa `--full` si quieres el cuerpo completo.
- `module update` y `module patch` son equivalentes.
