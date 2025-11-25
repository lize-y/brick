import time
import threading
from threading import Thread
from modelscope import AutoModelForCausalLM, AutoTokenizer
from transformers import TextIteratorStreamer

class LLMEngine:
    def __init__(self):
        self.model = None
        self.tokenizer = None
        self.is_loaded = False
        self._lock = threading.Lock()
        self.model_name = "Qwen/Qwen2.5-0.5B-Instruct"

    def load_model(self):
        """
        Load the Qwen model using modelscope.
        """
        with self._lock:
            if self.is_loaded:
                return
            print(f"Loading model {self.model_name}...")
            try:
                self.model = AutoModelForCausalLM.from_pretrained(
                    self.model_name,
                    torch_dtype="auto",
                    device_map="auto"
                )
                self.tokenizer = AutoTokenizer.from_pretrained(self.model_name)
                self.is_loaded = True
                print("Model loaded.")
            except Exception as e:
                print(f"Error loading model: {e}")
                raise e

    def generate_stream(self, prompt: str, max_tokens: int = 512):
        """
        Generator that yields tokens using TextIteratorStreamer.
        """
        if not self.is_loaded:
            raise RuntimeError("Model not loaded")

        print(f"Generating for prompt: {prompt}")
        
        system_prompt = "你是一个助手，帮助用户完成各种任务。"
        messages = [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": prompt}
        ]
        
        text = self.tokenizer.apply_chat_template(
            messages,
            tokenize=False,
            add_generation_prompt=True
        )
        
        model_inputs = self.tokenizer([text], return_tensors="pt").to(self.model.device)
        
        streamer = TextIteratorStreamer(self.tokenizer, skip_prompt=True, skip_special_tokens=True)
        
        generation_kwargs = dict(
            model_inputs,
            streamer=streamer,
            max_new_tokens=max_tokens
        )
        
        thread = Thread(target=self.model.generate, kwargs=generation_kwargs)
        thread.start()
        
        for new_text in streamer:
            yield new_text
