# Diagrama ER por módulos

El diagrama omite columnas de auditoría y tablas puente menores para mostrar las
autoridades principales. Las flechas representan FKs, no flujo de ejecución.

```mermaid
flowchart LR
  subgraph Identity[Identidad y autorización]
    U[users] --> C[user_credentials]
    U --> S[sessions]
    U --> UR[user_roles]
    R[roles] --> UR
    R --> RP[role_permissions]
    P[permissions] --> RP
    U --> TM[team_memberships]
    T[teams] --> TM
  end

  subgraph Configuration[Juego y concurso]
    G[games] --> GR[game_releases]
    A[artifacts] --> GR
    EV[evaluation_policy_versions] --> CT[contest_tasks]
    SP[scoring_policy_versions] --> CT
    CO[contests] --> D[contest_divisions]
    CO --> CT
    GR --> CT
    T --> CE[contest_entries]
    D --> CE
    CT --> TC[test_cases]
  end

  subgraph Code[Programa y build]
    CE --> SU[submissions]
    CT --> SU
    U --> SU
    A --> SU
    SU --> BA[build_attempts]
    A --> BA
  end

  subgraph Execution[Tanda y ejecución]
    CT --> EB[evaluation_batches]
    EB --> BR[batch_roster]
    SU --> BR
    BA --> BR
    EB --> M[matches]
    TC --> M
    BR --> MS[match_seats]
    M --> MS
    M --> MJ[match_jobs]
    M --> MA[match_attempts]
    MJ --> MA
    MA --> MR[match_seat_results]
    MS --> MR
    MA --> RE[match_replays]
  end

  subgraph Scoring[Juicio, score y publicación]
    SU --> J[judgements]
    EB --> J
    J --> JM[judgement_matches]
    M --> JM
    MS --> JM
    J --> RI[rejudge_items]
    RB[rejudge_batches] --> RI
    J --> SC[score_cells]
    CE --> SC
    CT --> SC
    SR[score_revisions] --> SC
    SR --> SW[score_rows]
    SR --> PUB[scoreboard_publications]
    PUB --> PR[publication_rows]
    PUB --> PC[publication_cells]
  end
```

Cadena temporal reproducible:

```mermaid
sequenceDiagram
  participant Team
  participant DB
  participant Worker
  participant Jury
  Team->>DB: submit_program (artefacto ready)
  Jury->>DB: sellar tanda y roster
  Jury->>DB: sellar partida, semilla y seats
  Worker->>DB: claim SKIP LOCKED + fencing N
  Worker->>DB: resultados por seat + replay
  Worker->>DB: accept attempt N
  Jury->>DB: juicio con evidencia match/seat
  DB->>DB: revisión de score completa
  Jury->>DB: publicación snapshot
```

