# Exam RAG Agent Foundation Progress

## 2026-07-08

- [x] Added deterministic Gaokao English reading query eval cases for structured exam context answerability.
- [x] Added reusable `EvaluateStructuredExamQuestionContext` summary with per-query `Passed`, `Score`, `MissingPhrases`, `ContextLabel`, and `SourceChunkIDs`.
- [x] Added `exam_question_context` query-only retrieval path so Agent tools can resolve natural-language exam questions to structured question-group context.
- [x] Added resolver-based `EvaluateExamContextRetrieval` to measure retrieval hit rate and answerability separately.
- [x] Added shared `internal/examrag` resolver for the production path: `HybridSearch -> candidate chunk IDs -> QuestionGroupDetail -> StructuredBundle`.
- [x] Wired `exam_question_context`, Agent `knowledge_search` exam enrichment, Chat pipeline exam enrichment, and retrieval eval adapters to the shared resolver path.
- [x] Added `ExamQuestionContextResolver.EvaluateRetrieval` so a real KB scope can be benchmarked with query cases using the same production resolver path.
- [x] Added reading-passage material enrichment before question-group draft validation.
  - Restores missed standalone headings, table titles, and introductory lines from neighboring source chunks.
  - Handles the observed `Upcoming Football Events` gap where model output referenced chunk 10 and 12 but skipped the heading in chunk 11.
  - Keeps question lines out of `material_text` so official question groups remain passage-first, question-second.
- [x] Added student-side `exam_learning_diagnosis` Agent tool on top of practice wrong-question data.
  - Reads the current session user and tenant from context.
  - Returns wrong-question review context with group metadata, stem, options, student answer, correct answer, explanation, review status, and review note.
  - Defaults to hiding mastered questions while reporting omitted-mastered count; supports `include_mastered`.
  - Registered in backend default tools, capability requirements, Agent runtime registry, and frontend Agent editor.
- [x] Extended the diagnosis layer to teacher/class scope with `exam_class_diagnosis`.
  - Reuses `ExamAnalyticsService` as the permission-enforced source of truth.
  - Aggregates frequent wrong questions only from each student's latest attempt per assignment.
  - Emits deterministic mastery trends, at-risk students, and compact Agent context.
  - Lists analyzable classes when class scope is ambiguous and masks not-found versus forbidden details from the model.

## 2026-07-10

- [x] Replaced synchronous question-group extraction HTTP calls with `exam:question_group_extract` Asynq tasks on the isolated `question` queue.
- [x] Registered the same worker in Redis-backed and Lite-mode task executors.
- [x] Added migration `000079` and persisted `queued/preflight/extracting/quality_check/completed/failed` progress on structuring tasks.
- [x] Added batch-level progress callbacks for the five-batch Gaokao math extraction strategy.
- [x] Added formula-heavy DOCX preflight warnings for WMF image references with PDF/VLM remediation guidance.
- [x] Added math draft quality checks for empty options, missing answers, missing sub-questions, missing figures, short stems, and missing explanations.
- [x] Added a blocking approval gate; live API verification found 3 blocking drafts and rejected an invalid approval with HTTP 400.
- [x] Added frontend polling, persisted progress display, preflight warnings, and per-draft quality reports.
- [x] Added subject-aware math RAG diagnostics and included question-group assets in structured RAG context.
- [x] Rebuilt Docker app/frontend images; database migrated from version 78 to 79 and all five core containers are healthy.
- [x] Live worker verification: HTTP returned 202 in 19 ms, loaded 29 chunks, and persisted batch 1/5 progress without panic.
- [x] Resolved the SiliconFlow balance block by switching the production test path to local Ollama: `bge-m3` for embeddings and `gemma3:12b` for structured extraction/summarization.
- [x] Implemented and deployed the teacher/class diagnosis Agent tool while the external model balance remains blocked.
- [x] Live class analytics verification: teacher request succeeded, the same request as a non-member student returned HTTP 403, and an empty class produced no fabricated wrong-question evidence.
- [x] Ran one end-to-end local-Ollama Agent conversation that combines `exam_class_diagnosis` with `exam_question_context`; the math extraction rerun is already covered by the 2026-07-13 acceptance below.

## 2026-07-10 Adaptive Practice Intervention

- [x] Added a shared `ExamInterventionService` for class and student practice recommendations over formal, readable question groups.
- [x] Added deterministic structured-signal scoring, stable ordering, explicit supplemental fallback, and mastered-question suppression.
- [x] Prevented class recommendations from using the teacher's personal latest practice attempt as class evidence.
- [x] Replaced the recent-100 assignment scan with a candidate-scoped distinct lookup across all published class assignments.
- [x] Added `exam_practice_recommendation` to Agent tools, capabilities, presets, and built-in agent configuration.
- [x] Added class/student HTTP recommendation endpoints and verified permission-denied mapping to HTTP 403.
- [x] Added the class analytics recommendation dialog, explicit include-published toggle, and manual assignment-form prefill.
- [x] Independent review completed; recommendation remains read-only and assignment creation still requires teacher confirmation.
- [x] Runtime verification after Docker rebuild: teacher 200, non-member student 403, student recommendation 200, and assignment publication from a recommendation.
- [x] Added stable structured-content identity matching for cross-space assignment clones; default recommendations exclude published sources, while include-published mode deduplicates source and clone and prefers the class-space copy.
- [x] Completed browser QA at 390x844 and 1440x900: mobile cards and desktop table render correctly, published A/D groups appear once, and assignment title/group prefill remains intact.
- [x] Removed the platform's narrow-screen horizontal overflow by using the existing collapsed navigation presentation and made the assignment form viewport-safe on mobile.
- [x] Closed independent-review findings with latest-request-wins recommendation loading, stale-result clearing, legacy member identifier fallback, and runtime tests for both behaviors.

## 2026-07-13 Local Math Structuring Acceptance

- [x] Replaced the fixed five-batch assumption with question-boundary-aware batching; the formal 2025 Gaokao math paper completed 12/12 batches.
- [x] Added deterministic source recovery for complete stems, A-D options, official answers, sub-question numbering, and WMF/image references while rejecting answer-page pseudo questions and model-invented sub-questions.
- [x] Completed task `0c26753c-cc13-49b8-bffc-aaf47c9d6f53` with 19/19 top-level groups, 28 formal questions/sub-questions, 151 assets, 0 duplicate numbers, and 0 blocking quality errors.
- [x] Approved all 19 drafts and verified the formal question-bank APIs and browser page from `Question 1` through `Question 19`.
- [x] Re-ran related Go tests and `go vet`, all 179 frontend tests, and the Vite production build.

## 2026-07-14 Local RAG Agent Acceptance

- [x] Switched all four existing knowledge bases to the local Ollama `bge-m3` embedding model with 1024-dimensional vectors.
- [x] Created `gemma3:12b-tools` from the existing local Gemma 3 12B weights with a Gemma-native tool-calling template; no duplicate model-weight download was required.
- [x] Configured the `班级学情诊断助手` Agent with `exam_class_diagnosis` and `exam_question_context` and bound it to the formal Gaokao English knowledge base.
- [x] Completed a real Agent conversation in the order `exam_class_diagnosis -> exam_question_context -> final answer`; both tool results succeeded and the answer included class evidence, questions 21-23, and teaching suggestions.
- [x] Verified the persisted conversation in the browser: the expanded tool timeline displayed both tool calls and the final Chinese answer, using `Google Gemma 3 12B Tools (Ollama)`.
- [x] Re-ran the scoped Go tests and `go vet`, all 179 frontend tests, and the Vite production build. The three pre-existing DuckDB Excel tests remain environment-limited because their online extension installation has no timeout; the other 100 `internal/agent/tools` tests passed.
