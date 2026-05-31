import logging
from concurrent import futures
import grpc
import sys
import os
from dotenv import load_dotenv

# Load environment variables from .env file
load_dotenv()

# Add current directory to path so generated protos can be found
sys.path.append(os.path.dirname(os.path.abspath(__file__)))

from analyzers.prompt_injection import PromptAnalyzer, GeminiPromptAnalyzer

try:
    import analyzer_pb2
    import analyzer_pb2_grpc
except ImportError:
    print("Error: Proto files not found. Please run 'make generate-proto' first.", file=sys.stderr)
    sys.exit(1)

class AIAnalyzerServicer(analyzer_pb2_grpc.AIAnalyzerServicer):
    """
    Implements the gRPC interface. Uses dependency injection to receive
    the specific analyzer implementation (Gemini).
    """
    def __init__(self, logger: logging.Logger, prompt_analyzer: PromptAnalyzer):
        self.logger = logger
        self.prompt_analyzer = prompt_analyzer

    def AnalyzeRequest(self, request, context):
        self.logger.info(f"Analyzing request: Project={request.project_id}, IP={request.client_ip}, Method={request.method}")
        
        risk_score = 0
        is_malicious = False
        reason = "Safe"
        flags = []

        # We only run prompt injection analysis on requests with bodies
        if request.body and request.method in ["POST", "PUT", "PATCH"]:
            self.logger.info("Executing Gemini AI Prompt Injection Analysis...")
            risk_score, is_malicious, reason, flags = self.prompt_analyzer.analyze(request.body)
            self.logger.info(f"Analysis Complete - Risk: {risk_score}, Malicious: {is_malicious}, Reason: {reason}")
        else:
            self.logger.info("Skipping analysis: No request body or safe HTTP method.")
        
        return analyzer_pb2.AnalyzeResponseMessage(
            risk_score=risk_score,
            block_recommended=is_malicious,
            reason=reason,
            flags=flags
        )

def serve():
    logging.basicConfig(
        level=logging.INFO, 
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
    logger = logging.getLogger("AegisAIEngine")

    # Dependency Injection: Instantiate Gemini Analyzer
    prompt_analyzer = GeminiPromptAnalyzer(logger)

    # Initialize gRPC server
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    analyzer_pb2_grpc.add_AIAnalyzerServicer_to_server(AIAnalyzerServicer(logger, prompt_analyzer), server)
    
    port = os.environ.get("PORT", "50051")
    server.add_insecure_port(f"[::]:{port}")
    
    logger.info(f"AI Engine gRPC server listening on port {port}...")
    server.start()
    server.wait_for_termination()

if __name__ == '__main__':
    serve()
