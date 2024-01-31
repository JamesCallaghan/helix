from flask import Flask, request, jsonify
import pprint
import sql
from embedding import getEmbedding
from chunks import parse_document
from utils import HttpException

app = Flask(__name__)

# curl -X POST -H "Content-Type: application/json" -d '{
#   "session_id": "123",
#   "interaction_id": "456",
#   "filename": "test.txt",
#   "document_id": "abc",
#   "document_group_id": "def",
#   "offset": 0,
#   "text": "hello world"
# }' http://localhost:5000/api/v1/rag/chunk
# this route will convert the text chunk into an embedding and then store it in the database
@app.route('/api/v1/rag/chunk', methods=['POST'])
def rag_insert_chunk():
  data = request.json
  sql.checkDocumentChunkData(data)
  data["embedding"] = getEmbedding(data["content"])
  id = sql.insertData(engine, data)
  result = sql.getRow(engine, id)
  pprint.pprint(result)
  return jsonify(result), 200

# curl -X POST -H "Content-Type: application/json" -d '{
#   "session_id": "123",
#   "prompt": "hello world"
# }' http://localhost:5000/api/v1/rag/prompt
# this will
#  * convert the prompt
#  * conduct a search on matching records (for that session)
#  * formulate a prompt that contains the context of the matching records
#  * return the prompt alongside the matching records (so we can show provenance of what was matched in the UI) 
@app.route('/api/v1/rag/prompt', methods=['POST'])
def rag_query():
  data = request.json
  prompt = data["prompt"]
  session_id = data["session_id"]
  if prompt is None or len(prompt) == 0:
    return jsonify({"error": "missing prompt"}), 400
  if session_id is None or len(session_id) == 0:
    return jsonify({"error": "missing session_id"}), 400
  promptEmbedding = getEmbedding(prompt)
  results = sql.queryPrompt(session_id, promptEmbedding)
  pprint.pprint(results)
  return jsonify({
    "ok": True,
    "results": results,
  }), 200

@app.route('/api/v1/extract', methods=['POST'])
def extract_file():
  if 'url' not in request.json:
    return jsonify({"error": "No 'url' field in the request"}), 400
  
  url = request.json['url']

  print("-------------------------------------------")
  print(f"converting URL: {url}")
  try:
    text = parse_document(url)
    print("-------------------------------------------")
    print(f"converted URL: {url} - length: {len(text)}")

    return jsonify({
      "text": text,
    }), 200
  
  except HttpException as e:
    print("-------------------------------------------")
    print(f"error URL: {url} - {str(e)}")
    return str(e), e.status_code
  except Exception as e:
    print("-------------------------------------------")
    print(f"error URL: {url} - {str(e)}")
    return str(e), 500

if __name__ == '__main__':
  app.run(debug=True, port=5000, host='0.0.0.0')
