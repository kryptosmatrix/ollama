# Continued source consultation — finding dispositions

These are three messages in one continued DeepSeek source/log consultation, not three independent judges or formal promotion rounds. Requested model deepseek-v4.1-flash:cloud; each served deepseek-v4.1-flash. No reviewer-directed execution occurred. Full requests/responses/reports remain unchanged.

| Finding | Disposition and evidence |
|---|---|
| One-token inference is not a free preflight tokenizer | Accepted; the earlier cloud measurements really inferred. The selected design rejects that budget mechanism, not ordinary cloud use elsewhere. |
| Native non-inferencing counter had not been located | Resolved in source at pinned b10091: /v1/chat/completions/input_tokens and its parser/count body exist. Runtime availability remains unverified. |
| A new header would force an older daemon to reject strict requests | Disproved: an unknown header may be ignored. Versioned capability/ingress remains open. |
| Public Go Tokenize matches final native dispatch | Disproved as a general source claim: add_special=false versus native true,true. A model-specific undercount was not measured. |
| Equal request JSON entails identical later template output | Disproved by clock-dependent template inputs. Preview does not reserve or prove dispatch. |
| Only a new pre-task/pre-queue hook is sound | Disproved. Existing actual-token/selected-slot guard at server-context.cpp:3068–3150 is the selected extension site before prompt evaluation; existing input-only checks and thresholds are not the future output-reserve comparison. |
| Go ContextLength is the actual selected capacity | Insufficient: it returns options.NumCtx; native slot construction can cap capacity to training context at :1250–1255. |
| The draft claims the count endpoint proves queued input | Rejected against the draft: section6.2 expressly separates preview and dispatch. Its unnecessarily early queue timing was nevertheless corrected to slot-start pre-evaluation. |
| Native media has no equivalent slot because has_mtmd disables cache reuse | Rejected: disabling cache reuse is not bypassing slot representation or the preceding guard. The supplied :3068–3150 flow precedes the cache branch. Media accounting still needs real qualification. |
| Reference fixtures prove the production store/non-null binding serializer | Rejected: all three use null bindings and only verify reference standard-library bytes/hash. Actual production write/read/consumer and non-null-binding proof remain owed. |
| Version timeout proves no internal model operation occurred | Rejected: no model-specific probe was launched, but internal executable activity was not observed. |

The reviewer agrees that the selected existing slot guard is a source-supported extension site. This does not promote G1, qualify the native runtime, select an output constant, or close cloud/MLX/strict-ingress contracts. Reviewer prescriptions are not implementation authority.
