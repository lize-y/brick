import sys
import os

# Ensure src is in python path
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), '../')))

from ai_core.core.engine import LLMEngine
from ai_core.server.grpc_server import serve

def main():
    print("Initializing AI Core...")
    engine = LLMEngine()
    engine.load_model()
    
    print("Starting gRPC server...")
    serve(engine)

if __name__ == "__main__":
    main()
