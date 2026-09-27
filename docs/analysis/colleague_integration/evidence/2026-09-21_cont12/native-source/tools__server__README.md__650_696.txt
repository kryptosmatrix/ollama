650:     {"id": 164, "piece": [161]} // hex A1
651:   ]
652: }
653: ```
654: 
655: ### POST `/detokenize`: Convert tokens to text
656: 
657: *Options:*
658: 
659: `tokens`: Set the tokens to detokenize.
660: 
661: ### POST `/apply-template`: Apply chat template to a conversation
662: 
663: Uses the server's prompt template formatting functionality to convert chat messages to a single string expected by a chat model as input, but does not perform inference. Instead, the prompt string is returned in the `prompt` field of the JSON response. The prompt can then be modified as desired (for example, to insert "Sure!" at the beginning of the model's response) before sending to `/completion` to generate the chat response.
664: 
665: *Options:*
666: 
667: `messages`: (Required) Chat turns in the same format as `/v1/chat/completions`.
668: 
669: **Response format**
670: 
671: Returns a JSON object with a field `prompt` containing a string of the input messages formatted according to the model's chat template format.
672: 
673: ### POST `/embedding`: Generate embedding of a given text
674: 
675: > [!IMPORTANT]
676: >
677: > This endpoint is **not** OAI-compatible. For OAI-compatible client, use `/v1/embeddings` instead.
678: 
679: The same as [the embedding example](../embedding) does.
680: 
681: This endpoint also supports multimodal embeddings. See the documentation for the `/completions` endpoint for details on how to send a multimodal prompt.
682: 
683: *Options:*
684: 
685: `content`: Set the text to process.
686: 
687: `embd_normalize`: Normalization for pooled embeddings. Can be one of the following values:
688: ```
689:   -1: No normalization
690:    0: Max absolute
691:    1: Taxicab
692:    2: Euclidean/L2
693:   >2: P-Norm
694: ```
695: 
696: ### POST `/reranking`: Rerank documents according to a given query
