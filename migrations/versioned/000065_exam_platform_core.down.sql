-- Migration: 000065_exam_platform_core (down)

DROP TABLE IF EXISTS question_chunk_refs;
DROP TABLE IF EXISTS question_knowledge_points;
DROP TABLE IF EXISTS question_explanations;
DROP TABLE IF EXISTS question_answers;
DROP TABLE IF EXISTS question_options;
DROP TABLE IF EXISTS questions;
DROP TABLE IF EXISTS question_banks;
DROP TABLE IF EXISTS exam_class_members;
DROP TABLE IF EXISTS exam_classes;
DROP TABLE IF EXISTS exam_spaces;
DROP TABLE IF EXISTS knowledge_points;
DROP TABLE IF EXISTS question_types;
DROP TABLE IF EXISTS exam_subjects;
DROP TABLE IF EXISTS exam_domains;
