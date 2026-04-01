# Vocabulary Module — Domain Model

## Entity Relationship Review

### Aggregate Roots

| Aggregate | Root Entity | Owned Entities |
|-----------|-------------|----------------|
| Vocabulary | `Vocabulary` | `VocabularyMeaning`, `VocabularyExample` |
| Folder | `Folder` | *(none — references Vocabulary by ID)* |

### Reference Data (Standalone Entities)

`Language` → `Category` → `ProficiencyLevel` / `Topic` / `GrammarPoint`

These are lookup/taxonomy entities, not aggregate roots. Each has its own repository.

### Cross-Aggregate References (by ID only)

| From | To | Via |
|------|----|-----|
| Vocabulary | Language | `languageID` FK |
| Vocabulary | ProficiencyLevel | `proficiencyLevelID` FK |
| Vocabulary | Topic | `vocabulary_topics` junction (M:M) |
| Vocabulary | GrammarPoint | `vocabulary_grammar_points` junction (M:M) |
| VocabularyMeaning | Language | `languageID` FK (target language) |
| Folder | User (auth module) | `userID` FK |
| Folder | Language | `languageID` FK |
| Folder | Vocabulary | `folder_vocabularies` junction (M:M) |
| GrammarPoint | ProficiencyLevel | `proficiencyLevelID` FK (optional) |

### Design Notes

1. **Potential inconsistency**: `Vocabulary` references both `LanguageID` and `ProficiencyLevelID` directly, but `ProficiencyLevel` belongs to `Category` which belongs to `Language`. No domain-level validation ensures the `ProficiencyLevelID` belongs to the correct language chain. Consider validating at the use case layer.

2. **Same concern for M:M**: `Vocabulary ↔ Topic` and `Vocabulary ↔ GrammarPoint` — Topic/GrammarPoint belong to a Category scoped to a Language, but there is no constraint ensuring they match the Vocabulary's language.

3. **Folder is user-scoped, Vocabulary is system-wide**: Correct design — vocabularies are shared content, folders are personal collections.

---

## 1. Domain Model Diagram

```mermaid
classDiagram
    direction TB

    class Vocabulary {
        <<Aggregate Root>>
        VocabularyID id
        LanguageID languageID
        ProficiencyLevelID proficiencyLevelID
        string word
        string phonetic
        string audioURL
        string imageURL
        int frequencyRank
        map~string,any~ metadata
        []TopicID topicIDs
        []GrammarPointID grammarPointIDs
    }

    class VocabularyMeaning {
        <<Entity>>
        MeaningID id
        VocabularyID vocabularyID
        LanguageID languageID
        string meaning
        string wordType
        bool isPrimary
        int offset
    }

    class VocabularyExample {
        <<Entity>>
        ExampleID id
        MeaningID meaningID
        string sentence
        string phonetic
        map~string,string~ translations
        string audioURL
        int offset
    }

    class Folder {
        <<Aggregate Root>>
        FolderID id
        UserID userID
        LanguageID languageID
        string name
        string description
    }

    class Language {
        <<Entity>>
        LanguageID id
        string code
        string nameEN
        string nameNative
        bool isActive
        map~string,any~ config
    }

    class Category {
        <<Entity>>
        CategoryID id
        LanguageID languageID
        *int prepCategoryID
        string code
        string name
        bool isPublic
    }

    class ProficiencyLevel {
        <<Entity>>
        ProficiencyLevelID id
        CategoryID categoryID
        *int prepLevelID
        string code
        string name
        float64 target
        string displayTarget
        int offset
    }

    class Topic {
        <<Entity>>
        TopicID id
        CategoryID categoryID
        string slug
        map~string,string~ names
        int offset
    }

    class GrammarPoint {
        <<Entity>>
        GrammarPointID id
        CategoryID categoryID
        ProficiencyLevelID proficiencyLevelID
        string code
        string pattern
        map~string,any~ examples
        map~string,any~ rule
        map~string,any~ commonMistakes
    }

    Vocabulary "1" *-- "1..*" VocabularyMeaning : owns
    VocabularyMeaning "1" *-- "0..*" VocabularyExample : owns
    Vocabulary "M" -- "N" Topic : vocabulary_topics
    Vocabulary "M" -- "N" GrammarPoint : vocabulary_grammar_points
    Vocabulary "M" --> "1" Language : languageID
    Vocabulary "M" --> "1" ProficiencyLevel : proficiencyLevelID
    VocabularyMeaning "M" --> "1" Language : languageID

    Folder "M" -- "N" Vocabulary : folder_vocabularies
    Folder "M" --> "1" Language : languageID

    Language "1" <-- "M" Category : languageID
    Category "1" <-- "M" ProficiencyLevel : categoryID
    Category "1" <-- "M" Topic : categoryID
    Category "1" <-- "M" GrammarPoint : categoryID
    GrammarPoint "M" --> "0..1" ProficiencyLevel : proficiencyLevelID
```

---

## 2. Aggregate Diagram

```mermaid
graph TB
    subgraph vocab_agg["Vocabulary Aggregate"]
        direction TB
        V["<b>Vocabulary</b><br/><i>Aggregate Root</i>"]
        VM["<b>VocabularyMeaning</b><br/><i>Entity</i>"]
        VE["<b>VocabularyExample</b><br/><i>Entity</i>"]
        V -->|"owns 1..*"| VM
        VM -->|"owns 0..*"| VE
    end

    subgraph folder_agg["Folder Aggregate"]
        direction TB
        F["<b>Folder</b><br/><i>Aggregate Root</i>"]
    end

    subgraph ref_data["Reference Data - Standalone Entities"]
        direction TB
        L["<b>Language</b>"]
        C["<b>Category</b>"]
        PL["<b>ProficiencyLevel</b>"]
        T["<b>Topic</b>"]
        GP["<b>GrammarPoint</b>"]
        L --> C
        C --> PL
        C --> T
        C --> GP
        GP -.->|"optional"| PL
    end

    V -.->|"LanguageID"| L
    V -.->|"ProficiencyLevelID"| PL
    V -.->|"TopicIDs M:M"| T
    V -.->|"GrammarPointIDs M:M"| GP
    VM -.->|"LanguageID"| L

    F -.->|"UserID"| ext_user["User<br/><i>Auth module</i>"]
    F -.->|"LanguageID"| L
    F -.->|"VocabularyIDs M:M"| V

    style vocab_agg fill:#e3f2fd,stroke:#1565c0,stroke-width:2px
    style folder_agg fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px
    style ref_data fill:#f5f5f5,stroke:#757575,stroke-width:1px
```

> Dashed lines = reference by ID (cross-aggregate). Solid lines = ownership within aggregate.

---

## 3. Sequence Diagram — CreateVocabulary

```mermaid
sequenceDiagram
    actor Client
    participant H as VocabularyHandler
    participant UC as VocabularyCommandUseCase
    participant V as Vocabulary Domain
    participant Repo as VocabularyRepository

    Client->>H: POST /api/vocabularies (JSON)
    H->>H: Bind and validate request DTO

    H->>UC: CreateVocabulary(ctx, dto)
    UC->>V: NewVocabularyFromParams(params)
    V->>V: Validate word and meanings
    alt Validation fails
        V-->>UC: domain error
        UC-->>H: AppError 422
        H-->>Client: success false, error
    end
    V-->>UC: Vocabulary entity

    UC->>Repo: Create(ctx, vocabulary)
    alt DB error
        Repo-->>UC: raw error
        UC-->>H: AppError 500
        H-->>Client: success false, error
    end
    Repo-->>UC: nil

    opt TopicIDs provided
        UC->>Repo: SetTopics(ctx, vocabID, topicIDs)
    end
    opt GrammarPointIDs provided
        UC->>Repo: SetGrammarPoints(ctx, vocabID, grammarPointIDs)
    end

    UC-->>H: VocabularyResponse DTO
    H-->>Client: success true, data, meta
```

---

## 4. Sequence Diagram — Folder AddVocabulary

```mermaid
sequenceDiagram
    actor Client
    participant H as FolderHandler
    participant UC as FolderCommandUseCase
    participant FRepo as FolderRepository
    participant VRepo as VocabularyRepository

    Client->>H: POST /api/folders/:id/vocabularies
    H->>H: Bind request, extract userID from JWT

    H->>UC: AddVocabulary(ctx, folderID, vocabID, userID)

    UC->>FRepo: FindByID(ctx, folderID)
    alt Folder not found
        FRepo-->>UC: nil, nil
        UC-->>H: AppError 404
        H-->>Client: success false, error
    end
    FRepo-->>UC: Folder entity

    UC->>UC: Check folder.UserID == userID
    alt Not owner
        UC-->>H: AppError 403
        H-->>Client: success false, error
    end

    UC->>VRepo: FindByID(ctx, vocabID)
    alt Vocab not found
        VRepo-->>UC: nil, nil
        UC-->>H: AppError 404
    end

    UC->>FRepo: AddVocabulary(ctx, folderID, vocabID)
    FRepo-->>UC: nil
    UC-->>H: success
    H-->>Client: success true, meta
```
