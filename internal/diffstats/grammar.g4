// SPDX-License-Identifier: Apache-2.0
// Unified Diff Grammar — ANTLR4 format for git diff output.
// This grammar defines the lexical and syntactic structure of unified diffs
// produced by `git diff`, `diff -u`, and similar tools.

grammar UnifiedDiff;

// ── Parser Rules ────────────────────────────────────────────────────────────

diff          : file+ EOF ;

file          : header metadata* body ;

header        : DIFF ;

metadata      : INDEX_HDR
              | NEW_FILE_MODE
              | DELETED_FILE_MODE
              | OLD_MODE
              | NEW_MODE
              | RENAME_FROM
              | RENAME_TO
              | SIMILARITY
              | BINARY_HDR
              ;

body          : (OLD_FILE NEW_FILE)? hunk+ ;

hunk          : HUNK_HDR hunkLine* ;

hunkLine      : CONTEXT | ADDITION | DELETION | NO_NEWLINE_EOF ;

// ── Lexer Tokens ────────────────────────────────────────────────────────────

DIFF          : 'diff --git a/' .+? ' b/' .+? '\n' ;

INDEX_HDR     : 'index ' [0-9a-f]+ '..' [0-9a-f]+ ' ' [0-9]+ '\n' ;

NEW_FILE_MODE : 'new file mode ' [0-9]+ '\n' ;

DELETED_FILE_MODE : 'deleted file mode ' [0-9]+ '\n' ;

OLD_MODE      : 'old mode ' [0-9]+ '\n' ;

NEW_MODE      : 'new mode ' [0-9]+ '\n' ;

RENAME_FROM   : 'rename from ' .+? '\n' ;

RENAME_TO     : 'rename to ' .+? '\n' ;

SIMILARITY    : 'similarity index ' [0-9]+ '%' '\n' ;

BINARY_HDR    : 'Binary files ' .+? ' differ\n' ;

OLD_FILE      : '--- ' .+? '\n' ;

NEW_FILE      : '+++ ' .+? '\n' ;

HUNK_HDR      : '@@ -' [0-9]+ (',' [0-9]+)? ' +' [0-9]+ (',' [0-9]+)? ' @@' .*? '\n' ;

CONTEXT       : ' ' .*? '\n' ;

ADDITION      : '+' .*? '\n' ;

DELETION      : '-' .*? '\n' ;

NO_NEWLINE_EOF : '\\ No newline at end of file\n' ;
