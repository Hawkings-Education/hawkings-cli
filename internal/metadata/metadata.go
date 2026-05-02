package metadata

type Flag struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type Command struct {
	Path         string   `json:"path"`
	Summary      string   `json:"summary"`
	Method       string   `json:"method,omitempty"`
	Endpoint     string   `json:"endpoint,omitempty"`
	RequiresAuth bool     `json:"requires_auth"`
	Output       string   `json:"output"`
	Flags        []Flag   `json:"flags,omitempty"`
	Notes        []string `json:"notes,omitempty"`
}

type Entity struct {
	Name      string   `json:"name"`
	Summary   string   `json:"summary"`
	Endpoint  string   `json:"endpoint,omitempty"`
	Parent    string   `json:"parent,omitempty"`
	Children  []string `json:"children,omitempty"`
	KeyFields []string `json:"key_fields,omitempty"`
	Expanders []string `json:"expanders,omitempty"`
	Commands  []string `json:"commands,omitempty"`
	Notes     []string `json:"notes,omitempty"`
}

type HierarchyNode struct {
	Name        string          `json:"name"`
	Cardinality string          `json:"cardinality,omitempty"`
	Summary     string          `json:"summary"`
	Commands    []string        `json:"commands,omitempty"`
	Notes       []string        `json:"notes,omitempty"`
	Children    []HierarchyNode `json:"children,omitempty"`
}

type Model struct {
	Name       string          `json:"name"`
	Version    int             `json:"version"`
	Principles []string        `json:"principles"`
	Hierarchy  []HierarchyNode `json:"hierarchy"`
	Entities   []Entity        `json:"entities"`
	Commands   []Command       `json:"commands"`
}

func Catalog() []Command {
	return CommandsCatalog()
}

func ModelCatalog() Model {
	return Model{
		Name:    "hawkings",
		Version: 1,
		Principles: []string{
			"JSON estable por defecto fuera de TTY; tabla compacta para uso interactivo.",
			"La estructura se descubre por introspeccion del propio CLI mediante describe.",
			"El contenido grande se recupera con comandos explicitos y truncado por defecto.",
			"Program status es una pista, no una garantia; valida syllabus y courses por presencia real.",
		},
		Hierarchy: HierarchyCatalog(),
		Entities:  EntitiesCatalog(),
		Commands:  CommandsCatalog(),
	}
}

func HierarchyCatalog() []HierarchyNode {
	return []HierarchyNode{
		{
			Name:        "program",
			Cardinality: "1",
			Summary:     "Unidad raiz de navegacion. Puede tener syllabus y zero o mas courses.",
			Commands:    []string{"program get", "program tree", "program syllabus", "program courses", "program config"},
			Notes: []string{
				"El campo status no garantiza por si solo que existan courses.",
				"Para validar structure real usa program get, program tree o program courses.",
			},
			Children: []HierarchyNode{
				{
					Name:        "course",
					Cardinality: "0..n",
					Summary:     "Curso derivado del programa. Puede tener modules directos y sections.",
					Commands:    []string{"program courses", "course get", "course sections", "course modules", "course module-status"},
					Notes: []string{
						"Para ver la estructura completa usa with[]=courseModules y with[]=courseSectionsModules.",
					},
					Children: []HierarchyNode{
						{
							Name:        "section",
							Cardinality: "0..n",
							Summary:     "Agrupa modules dentro de un course.",
							Commands:    []string{"course sections", "program tree"},
							Children: []HierarchyNode{
								{
									Name:        "module",
									Cardinality: "0..n",
									Summary:     "Unidad de contenido o actividad. Puede vivir a nivel course o section.",
									Commands:    []string{"course modules", "module create", "module get", "module content"},
									Children: []HierarchyNode{
										{
											Name:        "content",
											Cardinality: "0..n",
											Summary:     "Contenido fisico del modulo. El texto suele estar en course_contents[].file.contents.",
											Commands:    []string{"module content", "module get --contents"},
											Notes: []string{
												"module content trunca por defecto para no saturar contexto.",
											},
										},
									},
								},
							},
						},
						{
							Name:        "module",
							Cardinality: "0..n",
							Summary:     "Modules de primer nivel del course, fuera de sections.",
							Commands:    []string{"course modules", "module create", "program tree", "module get", "module content"},
						},
					},
				},
			},
		},
	}
}

func EntitiesCatalog() []Entity {
	return []Entity{
		{
			Name:     "language",
			Summary:  "Dato de referencia para descubrir language_id y codigo locale.",
			Endpoint: "/language",
			KeyFields: []string{
				"id", "name", "code", "rtl",
			},
			Commands: []string{
				"language list",
			},
		},
		{
			Name:     "faculty",
			Summary:  "Dato de referencia para descubrir course_faculty_id en la platform activa.",
			Endpoint: "/course-faculty/{id}",
			KeyFields: []string{
				"id", "name", "code", "enabled", "course_area",
			},
			Expanders: []string{
				"courseArea",
			},
			Commands: []string{
				"faculty list",
			},
		},
		{
			Name:     "template",
			Summary:  "Template de programa con limites y actividades relacionadas.",
			Endpoint: "/course-program-template/{id}",
			KeyFields: []string{
				"id", "code", "name", "courses_min", "courses_max", "courses_hours_min", "courses_hours_max",
			},
			Commands: []string{
				"template list",
			},
			Notes: []string{
				"La respuesta incluye related con las actividades inyectadas por scope y position.",
			},
		},
		{
			Name:     "space",
			Summary:  "Contenedor transversal que agrupa programas, cursos y usuarios dentro de una platform.",
			Endpoint: "/space/{id}",
			Children: []string{"program"},
			KeyFields: []string{
				"id", "name", "remote_id", "description", "color", "personal", "enabled",
			},
			Expanders: []string{
				"user",
				"users",
				"usersCount",
				"coursePrograms",
				"courseProgramsCount",
			},
			Commands: []string{
				"space list",
				"space get",
				"space create",
				"space update",
				"space programs",
				"space assign-programs",
				"space users",
				"space assign-users",
				"space delete",
				"program set-spaces",
			},
			Notes: []string{
				"Relaciona programas por medio de /space/{id}/course-program o /course-program/{id}/space.",
				"space update es update total; el CLI hace GET previo para evitar perder campos.",
			},
		},
		{
			Name:     "program",
			Summary:  "Programa accesible para el usuario y la learning platform activa.",
			Endpoint: "/course-program/{id}",
			Children: []string{"course"},
			KeyFields: []string{
				"id", "name", "status", "enabled", "metadata", "language", "syllabus", "courses_count",
			},
			Expanders: []string{
				"courseProgramTemplate",
				"language",
				"user",
				"spaces",
				"courseFaculty",
				"coursesCount",
				"coursesSectionsModules",
				"coursesSectionsModulesContents",
			},
			Commands: []string{
				"program create",
				"program update",
				"program delete",
				"program set-spaces",
				"program set-courses",
				"program reorder-courses",
				"program add-course",
				"program remove-course",
				"program generate-syllabus",
				"program create-courses",
				"program image generate",
				"program image upload",
				"program list",
				"program get",
				"program tree",
				"program syllabus",
				"program config",
				"program courses",
				"program status-matrix",
			},
			Notes: []string{
				"status es orientativo; comprueba syllabus y courses por presencia real.",
				"coursesSectionsModules incluye courses, course modules y section modules.",
				"El listado usa courses_count para saber si hay courses sin expandirlos.",
				"Al borrar un program, el backend solo borra en cascada los courses exclusivos de ese program.",
				"La relacion program↔course tiene endpoint propio: /course-program/{id}/course.",
			},
		},
		{
			Name:     "course",
			Summary:  "Curso asociado a un program.",
			Endpoint: "/course/{id}",
			Parent:   "program",
			Children: []string{"section", "module"},
			KeyFields: []string{
				"id", "name", "status", "language", "course_sections", "course_modules",
			},
			Expanders: []string{
				"courseModules",
				"courseModulesContents",
				"courseSectionsModules",
				"courseSectionsModulesContents",
			},
			Commands: []string{
				"course list",
				"course create",
				"course image generate",
				"course image upload",
				"course get",
				"course sections",
				"course modules",
				"course module-status",
				"section generate-content",
				"section generate-activities",
			},
			Notes: []string{
				"GET /course/{id} sin with[] no devuelve el arbol completo.",
				"Un course puede tener modules directos ademas de sections.",
				"course create usa /course/bulk para crear el arbol y, si se le pasa --program, relaciona despues via /course-program/{id}/course.",
			},
		},
		{
			Name:     "scorm",
			Summary:  "Recurso SCORM creado por endpoint dedicado.",
			Endpoint: "/scorm",
			KeyFields: []string{
				"id", "name",
			},
			Commands: []string{
				"scorm create",
			},
			Notes: []string{
				"El CLI no envia user_id ni language_id aunque aparezcan en el payload de entrada.",
			},
		},
		{
			Name:     "section",
			Summary:  "Subdivision de un course que agrupa modules.",
			Parent:   "course",
			Children: []string{"module"},
			KeyFields: []string{
				"id", "name", "order", "course_modules",
			},
			Commands: []string{
				"course sections",
				"section generate-content",
				"section generate-activities",
				"program tree",
			},
			Notes: []string{
				"La lectura de estructura llega embebida en course o program tree; la generacion bulk se expone con section generate-*.",
			},
		},
		{
			Name:     "module",
			Summary:  "Unidad de aprendizaje, actividad o referencia.",
			Endpoint: "/course-module/{id}",
			Parent:   "course|section",
			Children: []string{"content", "activity"},
			KeyFields: []string{
				"id", "name", "type", "order", "status", "metadata", "course_contents", "activity",
			},
			Expanders: []string{
				"courseContents",
				"activity",
			},
			Commands: []string{
				"course modules",
				"module create",
				"module get",
				"module content",
				"module activity",
				"module set-content",
				"module set-activity",
				"module update",
				"module patch",
				"module generate-content",
				"module generate-activity",
				"module approve",
			},
			Notes: []string{
				"module get es ligero por defecto y solo lista content items.",
				"module content hace la lectura de file.contents con truncado por defecto.",
				"module create calcula order automaticamente cuando no se le pasa.",
				"module set-content permite escribir markdown manual sin pasar por el generador de contenido del modulo.",
				"module activity y module set-activity leen y actualizan la activity asociada a modulos type=activity.",
			},
		},
		{
			Name:     "activity",
			Summary:  "Actividad estructurada asociada a un module type=activity.",
			Endpoint: "/activity/{uuid|id}",
			Parent:   "module",
			KeyFields: []string{
				"id", "uuid", "type", "title", "status", "description", "content",
			},
			Expanders: []string{
				"activityQuestions",
				"courseModules",
			},
			Commands: []string{
				"module activity",
				"module set-activity",
				"module generate-activity",
				"section generate-activities",
			},
			Notes: []string{
				"El module expone la relation singular activity con with[]=activity.",
				"PATCH /activity/{uuid|id} exige title, description y content; el CLI lee la activity actual para preservar los campos omitidos.",
			},
		},
		{
			Name:    "content",
			Summary: "Contenido almacenado bajo un module, normalmente markdown o fichero asociado.",
			Parent:  "module",
			KeyFields: []string{
				"id", "name", "type", "file.id", "file.mime", "file.size", "file.contents",
			},
			Commands: []string{
				"module get",
				"module content",
				"module set-content",
				"content approve",
				"content delete",
			},
			Notes: []string{
				"El texto grande suele vivir en file.contents y puede ocupar miles de caracteres.",
				"Si hay varios contents, module content permite seleccionar por --content-id.",
				"module set-content crea o actualiza el course-content y luego puede marcar el module como processed.",
				"La aprobacion del contenido se persiste en approved_at del course-module.",
			},
		},
	}
}

func CommandsCatalog() []Command {
	return []Command{
		{
			Path:         "config show",
			Summary:      "Muestra la configuracion resuelta y sus fuentes.",
			RequiresAuth: false,
			Output:       "json|table",
		},
		{
			Path:         "auth whoami",
			Summary:      "Muestra el usuario autenticado y la learning platform activa.",
			Method:       "GET",
			Endpoint:     "/profile",
			RequiresAuth: true,
			Output:       "json|table",
		},
		{
			Path:         "platform list",
			Summary:      "Lista las learning platforms accesibles para el usuario autenticado.",
			Method:       "GET",
			Endpoint:     "/profile/learning-platform",
			RequiresAuth: true,
			Output:       "json|table",
		},
		{
			Path:         "language list",
			Summary:      "Lista los idiomas disponibles para descubrir language_id y code.",
			Method:       "GET",
			Endpoint:     "/language",
			RequiresAuth: true,
			Output:       "json|table",
		},
		{
			Path:         "faculty list",
			Summary:      "Lista facultades accesibles para descubrir course_faculty_id.",
			Method:       "GET",
			Endpoint:     "/course-faculty",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--limit", Type: "int", Description: "Limite por pagina."},
				{Name: "--page", Type: "int", Description: "Pagina a solicitar."},
				{Name: "--all", Type: "bool", Description: "Recorre todas las paginas y devuelve todos los resultados."},
				{Name: "--search", Type: "string", Description: "Texto de busqueda."},
				{Name: "--with", Type: "stringArray", Description: "Relaciones extra via with[]."},
			},
		},
		{
			Path:         "template list",
			Summary:      "Lista templates de programa para descubrir course_program_template_id.",
			Method:       "GET",
			Endpoint:     "/course-program-template",
			RequiresAuth: true,
			Output:       "json|table",
		},
		{
			Path:         "space list",
			Summary:      "Lista spaces accesibles para el usuario activo.",
			Method:       "GET",
			Endpoint:     "/space",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--limit", Type: "int", Description: "Limite por pagina."},
				{Name: "--page", Type: "int", Description: "Pagina a solicitar."},
				{Name: "--all", Type: "bool", Description: "Recorre todas las paginas y devuelve todos los resultados."},
				{Name: "--search", Type: "string", Description: "Texto de busqueda."},
				{Name: "--enabled", Type: "string", Description: "Filtra por enabled=true|false."},
				{Name: "--personal", Type: "string", Description: "Filtra por personal=true|false."},
				{Name: "--with", Type: "stringArray", Description: "Relaciones extra via with[]."},
			},
		},
		{
			Path:         "space get",
			Summary:      "Muestra el detalle de un space y, opcionalmente, sus programas asociados.",
			Method:       "GET",
			Endpoint:     "/space/{id}",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--with", Type: "stringArray", Description: "Relaciones extra via with[]."},
			},
		},
		{
			Path:         "space create",
			Summary:      "Crea un space a partir de un payload JSON.",
			Method:       "POST",
			Endpoint:     "/space",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--json", Type: "string", Description: "Payload JSON inline."},
				{Name: "--json-file", Type: "string", Description: "Ruta a un fichero JSON."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra el payload sin enviar peticiones."},
			},
		},
		{
			Path:         "space update",
			Summary:      "Hace GET del space, merge del patch JSON y PATCH del recurso completo.",
			Method:       "PATCH",
			Endpoint:     "/space/{id}",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--json", Type: "string", Description: "Patch JSON inline."},
				{Name: "--json-file", Type: "string", Description: "Ruta a un fichero JSON."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra el payload final sin enviarlo."},
			},
		},
		{
			Path:         "space programs",
			Summary:      "Lista los programas asociados a un space.",
			Method:       "GET",
			Endpoint:     "/space/{id}/course-program",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--with", Type: "stringArray", Description: "Relaciones extra via with[]."},
			},
		},
		{
			Path:         "space assign-programs",
			Summary:      "Asigna programas a un space mediante selected, add, remove o all.",
			Method:       "PATCH",
			Endpoint:     "/space/{id}/course-program",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--selected", Type: "intSlice", Description: "Lista exacta de program IDs asignados."},
				{Name: "--add", Type: "intSlice", Description: "Program IDs a anadir."},
				{Name: "--remove", Type: "intSlice", Description: "Program IDs a quitar."},
				{Name: "--all", Type: "intSlice", Description: "Program IDs usados como conjunto base de operacion."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra el payload sin enviar peticiones."},
			},
		},
		{
			Path:         "space users",
			Summary:      "Lista los usuarios asociados a un space.",
			Method:       "GET",
			Endpoint:     "/space/{id}/user",
			RequiresAuth: true,
			Output:       "json|table",
		},
		{
			Path:         "space assign-users",
			Summary:      "Asigna usuarios a un space mediante selected, add, remove o all.",
			Method:       "PATCH",
			Endpoint:     "/space/{id}/user",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--selected", Type: "intSlice", Description: "Lista exacta de user IDs asignados."},
				{Name: "--add", Type: "intSlice", Description: "User IDs a anadir."},
				{Name: "--remove", Type: "intSlice", Description: "User IDs a quitar."},
				{Name: "--all", Type: "intSlice", Description: "User IDs usados como conjunto base de operacion."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra el payload sin enviar peticiones."},
			},
		},
		{
			Path:         "space delete",
			Summary:      "Elimina un space si no tiene programas asociados.",
			Method:       "DELETE",
			Endpoint:     "/space/{id}",
			RequiresAuth: true,
			Output:       "json",
			Flags: []Flag{
				{Name: "--dry-run", Type: "bool", Description: "Muestra la operacion sin enviar peticiones."},
			},
			Notes: []string{
				"El backend rechaza el borrado si el space tiene programas asociados.",
			},
		},
		{
			Path:         "program create",
			Summary:      "Crea un programa desde un payload JSON y opcionalmente asigna spaces.",
			Method:       "POST",
			Endpoint:     "/course-program",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--json", Type: "string", Description: "Payload JSON inline."},
				{Name: "--json-file", Type: "string", Description: "Ruta a un fichero JSON."},
				{Name: "--syllabus-file", Type: "string", Description: "Ruta a un JSON con el syllabus a inyectar en el payload."},
				{Name: "--space", Type: "intSlice", Description: "IDs de spaces a asignar tras la creacion."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra los payloads sin enviar peticiones."},
			},
			Notes: []string{
				"Pensado para seguir el flujo POST /course-program y luego POST /course-program/{id}/space.",
			},
		},
		{
			Path:         "program update",
			Summary:      "Hace PATCH /only del programa con solo los campos enviados.",
			Method:       "PATCH",
			Endpoint:     "/course-program/{id}/only",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--json", Type: "string", Description: "Patch JSON inline."},
				{Name: "--json-file", Type: "string", Description: "Ruta a un fichero JSON."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra el patch tal como se enviara a PATCH /only."},
			},
			Notes: []string{
				"El backend preserva el estado actual del program y de metadata; el cliente no hace GET previo ni merge local.",
			},
		},
		{
			Path:         "program delete",
			Summary:      "Elimina un programa mostrando la semantica real de borrado de courses asociados.",
			Method:       "DELETE",
			Endpoint:     "/course-program/{id}",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--dry-run", Type: "bool", Description: "Muestra la operacion sin enviar peticiones."},
			},
			Notes: []string{
				"Los courses compartidos con otros programs se mantienen.",
				"El backend solo elimina en cascada los courses ligados exclusivamente a ese program.",
			},
		},
		{
			Path:         "program set-spaces",
			Summary:      "Reemplaza la seleccion de spaces de un programa.",
			Method:       "POST",
			Endpoint:     "/course-program/{id}/space",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--space", Type: "intSlice", Description: "IDs de spaces seleccionados."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra el payload sin enviar peticiones."},
			},
		},
		{
			Path:         "program set-courses",
			Summary:      "Reemplaza la seleccion de courses asociados a un programa.",
			Method:       "POST",
			Endpoint:     "/course-program/{id}/course",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--course", Type: "intSlice", Description: "IDs de courses que deben quedar asociados."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra el payload sin enviar peticiones."},
			},
		},
		{
			Path:         "program reorder-courses",
			Summary:      "Reordena los courses de un programa con un payload JSON completo y validado contra el estado actual.",
			Method:       "POST",
			Endpoint:     "/course-program/{id}/course",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--json", Type: "string", Description: "Payload JSON inline."},
				{Name: "--json-file", Type: "string", Description: "Ruta a un fichero JSON."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra el payload validado sin enviar peticiones."},
			},
			Notes: []string{
				"Exige selected y order en el payload.",
				"Antes de mutar, el CLI comprueba que selected coincide exactamente con los courses actuales del programa.",
				"order debe cubrir todas las posiciones 1..N y selected debe venir ya ordenado segun ese mapa.",
			},
		},
		{
			Path:         "program add-course",
			Summary:      "Anade uno o varios courses a un programa sin tocar los ya asociados.",
			Method:       "POST",
			Endpoint:     "/course-program/{id}/course",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--course", Type: "intSlice", Description: "IDs de courses a anadir."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra el payload sin enviar peticiones."},
			},
		},
		{
			Path:         "program remove-course",
			Summary:      "Quita uno o varios courses de un programa sin tocar los demas.",
			Method:       "POST",
			Endpoint:     "/course-program/{id}/course",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--course", Type: "intSlice", Description: "IDs de courses a quitar."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra el payload sin enviar peticiones."},
			},
		},
		{
			Path:         "program generate-syllabus",
			Summary:      "Lanza la generacion de syllabus usando el context actual o uno proporcionado.",
			Method:       "POST",
			Endpoint:     "/course-program/{id}/syllabus/generate",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--json", Type: "string", Description: "Payload JSON inline."},
				{Name: "--json-file", Type: "string", Description: "Ruta a un fichero JSON."},
				{Name: "--force", Type: "bool", Description: "Regenera aunque ya exista syllabus."},
				{Name: "--context", Type: "string", Description: "Sobrescribe el context usado para generar."},
				{Name: "--syllabus-prompt", Type: "string", Description: "Sobrescribe el syllabus_prompt de esta generacion."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra el payload sin enviar peticiones."},
			},
			Notes: []string{
				"Si no pasas context, el backend reutiliza el context ya guardado en el programa.",
			},
		},
		{
			Path:         "program create-courses",
			Summary:      "Crea los courses a partir del syllabus ya almacenado en el programa.",
			Method:       "POST",
			Endpoint:     "/course-program/{id}/syllabus/course",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--force", Type: "bool", Description: "Pide al backend forzar la operacion si esa variante esta soportada."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra la operacion sin enviar peticiones."},
			},
			Notes: []string{
				"El endpoint usa el syllabus ya guardado en el programa; no hay que mandarlo en el body.",
				"Si falla con un 422 sobre algun campo interno como type, el origen suele estar en datos derivados del syllabus o de la template del programa.",
				"La operacion puede tardar minutos; usa --timeout alto en programas grandes.",
				"Un timeout del cliente no garantiza cancelacion en backend; comprueba program get o program courses antes de relanzar.",
				"Tras un timeout, el programa puede quedar en courses-creating y un nuevo intento puede devolver 422 si ya existen courses parciales.",
			},
		},
		{
			Path:         "program image generate",
			Summary:      "Genera con IA la imagen de portada de un program.",
			Method:       "POST",
			Endpoint:     "/course-program/{id}/image/generate",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--force", Type: "bool", Description: "Regenera aunque el program ya tenga imagen."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra la operacion sin enviar peticiones."},
			},
			Notes: []string{
				"Esta opcion pide al backend generar la portada con IA; no sube ningun archivo local.",
			},
		},
		{
			Path:         "program image upload",
			Summary:      "Sube manualmente un JPG o PNG como imagen de portada de un program.",
			Method:       "PATCH",
			Endpoint:     "/course-program/{id}",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--file", Type: "string", Description: "Ruta a un archivo .jpg, .jpeg o .png."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra la operacion sin subir el archivo."},
			},
			Notes: []string{
				"Esta opcion envia multipart/form-data con el archivo en el campo image.",
				"El CLI lee primero el program actual para preservar sus campos al hacer PATCH.",
			},
		},
		{
			Path:         "program list",
			Summary:      "Lista los programas visibles para el usuario segun rol y plataforma.",
			Method:       "GET",
			Endpoint:     "/course-program",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--limit", Type: "int", Description: "Limite por pagina."},
				{Name: "--page", Type: "int", Description: "Pagina a solicitar."},
				{Name: "--all", Type: "bool", Description: "Recorre todas las paginas y devuelve todos los resultados."},
				{Name: "--search", Type: "string", Description: "Texto de busqueda."},
				{Name: "--status", Type: "string", Description: "Filtra por estado del programa."},
				{Name: "--order-column", Type: "string", Description: "Se envia como order_column; soporta varias columnas separadas por ';'."},
				{Name: "--order-mode", Type: "string", Description: "Se envia como order_mode; permite prioridad custom para status y direccion por columna."},
				{Name: "--space-id", Type: "int", Description: "Filtra por membresia real en un space usando /space/{id}/course-program."},
				{Name: "--with", Type: "stringArray", Description: "Relaciones a incluir mediante with[]."},
			},
			Notes: []string{
				"Cuando usas --space-id, el CLI consulta el endpoint del space y aplica search/status/paginacion en cliente.",
				"Con --space-id, la ordenacion por status/name tambien se replica en cliente para mantener el mismo comportamiento.",
				"En JSON, el listado siempre devuelve data junto con page, pages y total; usa --all si necesitas todos los resultados en una sola salida.",
			},
		},
		{
			Path:         "program get",
			Summary:      "Muestra el detalle de un programa y evalua si realmente tiene syllabus o courses.",
			Method:       "GET",
			Endpoint:     "/course-program/{id}",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--with", Type: "stringArray", Description: "Relaciones extra via with[]."},
				{Name: "--with-courses", Type: "bool", Description: "Incluye coursesSectionsModules."},
			},
		},
		{
			Path:         "program tree",
			Summary:      "Imprime el arbol navegable program -> course -> section -> module.",
			Method:       "GET",
			Endpoint:     "/course-program/{id}?with[]=coursesSectionsModules",
			RequiresAuth: true,
			Output:       "json|table",
			Notes: []string{
				"Pensado para navegar estructura sin traer contenidos largos.",
			},
		},
		{
			Path:         "program syllabus",
			Summary:      "Muestra el syllabus almacenado de un programa.",
			Method:       "GET",
			Endpoint:     "/course-program/{id}",
			RequiresAuth: true,
			Output:       "json|table",
		},
		{
			Path:         "program config",
			Summary:      "Muestra la configuracion relevante de un programa.",
			Method:       "GET",
			Endpoint:     "/course-program/{id}",
			RequiresAuth: true,
			Output:       "json|table",
		},
		{
			Path:         "program courses",
			Summary:      "Lista los courses asociados a un programa y resume su estructura.",
			Method:       "GET",
			Endpoint:     "/course-program/{id}?with[]=coursesSectionsModules",
			RequiresAuth: true,
			Output:       "json|table",
		},
		{
			Path:         "program status-matrix",
			Summary:      "Agrupa programas por status y disponibilidad de courses.",
			Method:       "GET",
			Endpoint:     "/course-program?with[]=coursesCount",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--limit", Type: "int", Description: "Tamano de pagina para recorrer el listado completo."},
				{Name: "--samples", Type: "int", Description: "Cuantos program IDs de ejemplo guardar por status."},
			},
			Notes: []string{
				"Usa courses_count del listado; para syllabus hay que ir a program get o program syllabus.",
			},
		},
		{
			Path:         "course list",
			Summary:      "Lista los courses accesibles y permite buscar por texto libre.",
			Method:       "GET",
			Endpoint:     "/course",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--limit", Type: "int", Description: "Limite por pagina."},
				{Name: "--page", Type: "int", Description: "Pagina a solicitar."},
				{Name: "--all", Type: "bool", Description: "Recorre todas las paginas y devuelve todos los resultados."},
				{Name: "--search", Type: "string", Description: "Texto libre para buscar por titulo, remote_id o uuid."},
				{Name: "--status", Type: "string", Description: "Filtra por status."},
				{Name: "--with", Type: "stringArray", Description: "Relaciones extra via with[]."},
			},
			Notes: []string{
				"En JSON, el listado siempre devuelve data junto con page, pages y total; usa --all si necesitas todos los resultados en una sola salida.",
				"Sirve para descubrir IDs de cursos reutilizables antes de usar program add-course, set-courses o remove-course.",
			},
		},
		{
			Path:         "course create",
			Summary:      "Crea o actualiza un course completo via /course/bulk y, si se le pasa --program, lo relaciona despues con el endpoint dedicado.",
			Method:       "POST",
			Endpoint:     "/course/bulk",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--program", Type: "int", Description: "ID del program a asociar despues via /course-program/{id}/course."},
				{Name: "--json", Type: "string", Description: "Payload JSON inline."},
				{Name: "--json-file", Type: "string", Description: "Ruta a un fichero JSON."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra el payload final sin enviar peticiones."},
			},
			Notes: []string{
				"course_sections es obligatorio en /course/bulk.",
				"Los modules markdown necesitan course_contents salvo que lleven empty=true.",
				"El backend sincroniza el arbol: si omites sections/modules existentes, puede eliminarlos.",
				"El backend puede responder 200 con errores parciales embebidos; el CLI inspecciona la respuesta y falla.",
				"Si usas --program, el CLI hace una segunda llamada con add al endpoint /course-program/{id}/course para evitar depender de course_programs dentro del bulk.",
			},
		},
		{
			Path:         "course image generate",
			Summary:      "Genera con IA la imagen de portada de un course.",
			Method:       "POST",
			Endpoint:     "/course/{id}/image/generate",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--force", Type: "bool", Description: "Regenera aunque el course ya tenga imagen."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra la operacion sin enviar peticiones."},
			},
			Notes: []string{
				"Esta opcion pide al backend generar la portada con IA; no sube ningun archivo local.",
			},
		},
		{
			Path:         "course image upload",
			Summary:      "Sube manualmente un JPG o PNG como imagen de portada de un course.",
			Method:       "PATCH",
			Endpoint:     "/course/{id}",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--file", Type: "string", Description: "Ruta a un archivo .jpg, .jpeg o .png."},
				{Name: "--json", Type: "string", Description: "Payload JSON inline con los campos de update que se deben preservar."},
				{Name: "--json-file", Type: "string", Description: "Ruta a un JSON con los campos de update que se deben preservar."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra la operacion sin subir el archivo."},
			},
			Notes: []string{
				"Esta opcion envia multipart/form-data con el archivo en el campo image.",
				"El PATCH normal de course puede tocar campos que course get no devuelve; por eso el CLI exige --json o --json-file.",
				"El payload debe incluir al menos name y language_id, y tambien cualquier campo del course que quieras preservar.",
			},
		},
		{
			Path:         "scorm create",
			Summary:      "Crea un recurso SCORM con payload JSON saneado antes de POST /scorm.",
			Method:       "POST",
			Endpoint:     "/scorm",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--json", Type: "string", Description: "Payload JSON inline."},
				{Name: "--json-file", Type: "string", Description: "Ruta a un fichero JSON."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra el payload final sin enviar peticiones."},
			},
			Notes: []string{
				"El CLI elimina user_id y language_id del payload antes de enviarlo.",
				"Si el backend sigue devolviendo status legacy courses_created, el CLI lo muestra como courses-created.",
			},
		},
		{
			Path:         "course get",
			Summary:      "Muestra un course con su arbol de sections y modules.",
			Method:       "GET",
			Endpoint:     "/course/{id}?with[]=courseModules&with[]=courseSectionsModules",
			RequiresAuth: true,
			Output:       "json|table",
		},
		{
			Path:         "course sections",
			Summary:      "Lista las sections de un course.",
			Method:       "GET",
			Endpoint:     "/course/{id}?with[]=courseSectionsModules",
			RequiresAuth: true,
			Output:       "json|table",
		},
		{
			Path:         "course modules",
			Summary:      "Lista todos los modules de un course, incluidos los de sus sections.",
			Method:       "GET",
			Endpoint:     "/course/{id}?with[]=courseModules&with[]=courseSectionsModules",
			RequiresAuth: true,
			Output:       "json|table",
		},
		{
			Path:         "course module-status",
			Summary:      "Devuelve el status de todos los modules de un course.",
			Method:       "GET",
			Endpoint:     "/course/{id}/course-module/status",
			RequiresAuth: true,
			Output:       "json|table",
		},
		{
			Path:         "section generate-content",
			Summary:      "Lanza la generacion asincrona de contenido para todos los modules de una section.",
			Method:       "POST",
			Endpoint:     "/course-section/{id}/course-content/generate",
			RequiresAuth: true,
			Output:       "json",
			Flags: []Flag{
				{Name: "--research-enabled", Type: "bool", Description: "Activa research para la generacion."},
				{Name: "--research-provider", Type: "string", Description: "Proveedor de research: Parallel o Perplexity."},
				{Name: "--research-quality", Type: "string", Description: "Calidad de research: high, medium o fast."},
				{Name: "--research-instructions", Type: "string", Description: "Instrucciones especificas para el research."},
				{Name: "--research-id", Type: "intSlice", Description: "IDs de research existentes a reutilizar."},
				{Name: "--prompt-custom", Type: "string", Description: "Instrucciones de redaccion para todos los modulos de la section."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra el payload sin enviar peticiones."},
			},
		},
		{
			Path:         "section generate-activities",
			Summary:      "Lanza la generacion asincrona de actividades para una section completa.",
			Method:       "POST",
			Endpoint:     "/course-section/{id}/activity/generate",
			RequiresAuth: true,
			Output:       "json",
			Flags: []Flag{
				{Name: "--dry-run", Type: "bool", Description: "Muestra el payload sin enviar peticiones."},
			},
		},
		{
			Path:         "module get",
			Summary:      "Muestra un module y, opcionalmente, el inventario de contents sin traer el cuerpo completo.",
			Method:       "GET",
			Endpoint:     "/course-module/{id}?with[]=courseContents",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--contents", Type: "bool", Description: "Incluye file.contents completo; puede devolver mucho texto."},
			},
		},
		{
			Path:         "module content",
			Summary:      "Devuelve el contenido de un module con truncado por defecto y seleccion explicita del content.",
			Method:       "GET",
			Endpoint:     "/course-module/{id}?with[]=courseContents&contents=true",
			RequiresAuth: true,
			Output:       "json|table|raw",
			Flags: []Flag{
				{Name: "--content-id", Type: "int", Description: "Selecciona un content concreto si el modulo tiene varios."},
				{Name: "--max-chars", Type: "int", Description: "Maximo de caracteres devueltos cuando no se usa --full."},
				{Name: "--full", Type: "bool", Description: "Desactiva el truncado y devuelve el contenido completo."},
				{Name: "--raw", Type: "bool", Description: "Imprime solo el cuerpo de texto."},
			},
			Notes: []string{
				"Pensado para contexto grande: por defecto devuelve solo un fragmento mas metadata.",
			},
		},
		{
			Path:         "module activity",
			Summary:      "Lee la activity asociada a un module type=activity.",
			Method:       "GET + GET",
			Endpoint:     "/course-module/{id}?with[]=activity + /activity/{uuid|id}",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--raw-content", Type: "bool", Description: "Imprime solo activity.content como JSON."},
				{Name: "--questions", Type: "bool", Description: "Incluye activityQuestions en la lectura detallada."},
				{Name: "--course-modules", Type: "bool", Description: "Incluye courseModules relacionados en la activity."},
				{Name: "--max-chars", Type: "int", Description: "Maximo de caracteres de activity.content en salida table."},
				{Name: "--full", Type: "bool", Description: "No trunca activity.content en salida table."},
			},
			Notes: []string{
				"Primero lee el module con with[]=activity para resolver el uuid/id de la activity.",
			},
		},
		{
			Path:         "module create",
			Summary:      "Crea un module nuevo a nivel course o section y resuelve order automaticamente si no se indica.",
			Method:       "POST",
			Endpoint:     "/course-module",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--name", Type: "string", Description: "Nombre visible del module."},
				{Name: "--type", Type: "string", Description: "Tipo: markdown, activity, assignment o url."},
				{Name: "--course-id", Type: "int", Description: "ID del course para modulos a nivel curso."},
				{Name: "--section-id", Type: "int", Description: "ID de la section para modulos anidados."},
				{Name: "--order", Type: "int", Description: "Posicion deseada; si se omite, el CLI calcula max(order)+1."},
				{Name: "--status", Type: "string", Description: "Status inicial del module."},
				{Name: "--url", Type: "string", Description: "URL del module cuando el tipo lo requiera."},
				{Name: "--optional", Type: "bool", Description: "Marca el module como opcional."},
				{Name: "--evaluable", Type: "bool", Description: "Marca el module como evaluable."},
				{Name: "--position", Type: "string", Description: "Posicion logica before|after si quieres persistirla."},
				{Name: "--metadata-json", Type: "string", Description: "Objeto JSON inline para metadata."},
				{Name: "--metadata-file", Type: "string", Description: "Ruta a un fichero JSON para metadata."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra el payload resuelto sin enviar peticiones."},
			},
			Notes: []string{
				"Usa --course-id para modulos a nivel curso o --section-id para modulos dentro de una section.",
				"Si no indicas --order, el CLI lee el ambito elegido y calcula el siguiente order disponible.",
				"Si indicas un order ya ocupado, el backend desplaza los modulos siguientes.",
			},
		},
		{
			Path:         "module set-content",
			Summary:      "Escribe contenido manual en el course-content del module sin usar la generacion del modulo.",
			Method:       "POST|PATCH + PATCH",
			Endpoint:     "/course-content (+ optional /course-module/{id}/only)",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--file", Type: "string", Description: "Ruta a un fichero de texto o markdown."},
				{Name: "--content-file", Type: "string", Description: "Alias explicito de --file."},
				{Name: "--content", Type: "string", Description: "Contenido inline para escribir directamente."},
				{Name: "--name", Type: "string", Description: "Nombre del course-content; por defecto usa el del modulo."},
				{Name: "--mime", Type: "string", Description: "Mime del course-content; por defecto text/markdown para .md."},
				{Name: "--content-status", Type: "string", Description: "Status del course-content creado o actualizado."},
				{Name: "--module-status", Type: "string", Description: "Status a aplicar despues al module via PATCH /only."},
				{Name: "--content-id", Type: "int", Description: "Actualiza un course-content concreto si el modulo tiene varios."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra la operacion sin enviar peticiones."},
			},
			Notes: []string{
				"Crea un course-content si el module no tiene ninguno, o actualiza el primero/seleccionado si ya existe.",
				"Evita el endpoint POST /course-module/{id}/course-content/generate, pero el backend puede seguir calculando summary y metadatos derivados.",
			},
		},
		{
			Path:         "module set-activity",
			Summary:      "Actualiza la activity asociada a un module type=activity.",
			Method:       "GET + GET + PATCH",
			Endpoint:     "/course-module/{id}?with[]=activity + /activity/{uuid|id}",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--json", Type: "string", Description: "Patch JSON inline con title, description y/o content."},
				{Name: "--json-file", Type: "string", Description: "Ruta a un fichero JSON con el patch."},
				{Name: "--title", Type: "string", Description: "Nuevo title de la activity."},
				{Name: "--description", Type: "string", Description: "Nueva description de la activity."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra el payload resuelto sin enviar el PATCH."},
			},
			Notes: []string{
				"El payload puede ser parcial; el CLI completa title, description y content desde la activity actual.",
				"El backend valida content segun el type de la activity.",
			},
		},
		{
			Path:         "content approve",
			Summary:      "Aprueba o desaprueba el contenido de un module usando approved_at del course-module.",
			Method:       "PATCH",
			Endpoint:     "/course-module/{id}/boolean/approved_at",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--approved", Type: "bool", Description: "true para aprobar, false para desaprobar."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra la operacion sin enviar peticiones."},
			},
			Notes: []string{
				"Es un alias semantico de module approve.",
				"En Hawkings, el estado de aprobacion del contenido vive en approved_at del module.",
			},
		},
		{
			Path:         "content delete",
			Summary:      "Elimina un course-content por ID.",
			Method:       "DELETE",
			Endpoint:     "/course-content/{id}",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--dry-run", Type: "bool", Description: "Muestra la operacion sin enviar peticiones."},
			},
			Notes: []string{
				"El argumento content-id debe ser un entero positivo.",
			},
		},
		{
			Path:         "module update",
			Summary:      "Alias semantico de module patch para hacer PATCH /only sobre un module.",
			Method:       "PATCH",
			Endpoint:     "/course-module/{id}/only",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--json", Type: "string", Description: "Patch JSON inline."},
				{Name: "--json-file", Type: "string", Description: "Ruta a un fichero JSON."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra el payload sin enviar peticiones."},
			},
			Notes: []string{
				"CLI alias de module patch para quien piense en terminos de update.",
			},
		},
		{
			Path:         "module patch",
			Summary:      "Hace PATCH /only sobre un module con solo los campos enviados.",
			Method:       "PATCH",
			Endpoint:     "/course-module/{id}/only",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--json", Type: "string", Description: "Patch JSON inline."},
				{Name: "--json-file", Type: "string", Description: "Ruta a un fichero JSON."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra el payload sin enviar peticiones."},
			},
		},
		{
			Path:         "module generate-content",
			Summary:      "Lanza la generacion asincrona del contenido de un module.",
			Method:       "POST",
			Endpoint:     "/course-module/{id}/course-content/generate",
			RequiresAuth: true,
			Output:       "json",
			Flags: []Flag{
				{Name: "--research-enabled", Type: "bool", Description: "Activa research para la generacion."},
				{Name: "--research-provider", Type: "string", Description: "Proveedor de research: Parallel o Perplexity."},
				{Name: "--research-quality", Type: "string", Description: "Calidad de research: high, medium o fast."},
				{Name: "--research-instructions", Type: "string", Description: "Instrucciones especificas para el research."},
				{Name: "--research-id", Type: "intSlice", Description: "IDs de research existentes a reutilizar."},
				{Name: "--prompt-custom", Type: "string", Description: "Instrucciones de redaccion para este modulo."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra el payload sin enviar peticiones."},
			},
		},
		{
			Path:         "module generate-activity",
			Summary:      "Lanza la generacion asincrona de la activity de un module type=activity.",
			Method:       "POST",
			Endpoint:     "/course-module/{id}/activity/generate",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--async", Type: "bool", Description: "Genera en background; usa --async=false para esperar."},
				{Name: "--priority", Type: "string", Description: "Prioridad opcional; el backend acepta low."},
				{Name: "--force", Type: "bool", Description: "Fuerza la generacion si el module esta procesando."},
				{Name: "--cache", Type: "bool", Description: "Controla la cache de generacion; solo se envia si pasas este flag."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra el payload sin enviar peticiones."},
			},
			Notes: []string{
				"El backend exige que el module sea type=activity y que metadata.activity.type exista.",
			},
		},
		{
			Path:         "module approve",
			Summary:      "Aprueba o desaprueba un module via approved_at.",
			Method:       "PATCH",
			Endpoint:     "/course-module/{id}/boolean/approved_at",
			RequiresAuth: true,
			Output:       "json|table",
			Flags: []Flag{
				{Name: "--approved", Type: "bool", Description: "true para aprobar, false para desaprobar."},
				{Name: "--dry-run", Type: "bool", Description: "Muestra la operacion sin enviar peticiones."},
			},
		},
		{
			Path:         "describe",
			Summary:      "Expone la jerarquia, entidades y comandos del CLI en JSON estable.",
			RequiresAuth: false,
			Output:       "json",
			Notes: []string{
				"Admite describe, describe hierarchy, describe entity <name> y describe command <path...>.",
			},
		},
	}
}
