import grpc
from concurrent import futures
import time
import threading
import sys
import os

# Add the api directory to sys.path to handle the imports in generated files
sys.path.append(os.path.join(os.path.dirname(__file__), '../api'))

from ai_core.api import llm_pb2
from ai_core.api import llm_pb2_grpc
from ai_core.core.engine import LLMEngine

class LLMService(llm_pb2_grpc.LLMServiceServicer):
    def __init__(self, engine: LLMEngine, stop_event: threading.Event):
        self.engine = engine
        self.stop_event = stop_event

    def GenerateStream(self, request, context):
        try:
            for token in self.engine.generate_stream(request.prompt, request.max_tokens):
                yield llm_pb2.TokenChunk(token=token)
        except Exception as e:
            context.set_details(str(e))
            context.set_code(grpc.StatusCode.INTERNAL)
            return

    def StopServer(self, request, context):
        print(f"Stop requested: {request.reason}")
        self.stop_event.set()
        return llm_pb2.StopResponse(message="Server stopping...")

def serve(engine: LLMEngine, port: int = 50051):
    stop_event = threading.Event()
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    llm_pb2_grpc.add_LLMServiceServicer_to_server(LLMService(engine, stop_event), server)
    server.add_insecure_port(f'[::]:{port}')
    server.start()
    print(f"Server started on port {port}")
    
    # Wait for stop event
    stop_event.wait()
    print("Stopping server...")
    server.stop(grace=5).wait()
    print("Server stopped.")
