// AudioWorklet Processor for Gemini Live API
// Captures microphone input and sends audio chunks to main thread

class AudioProcessor extends AudioWorkletProcessor {
  constructor(options) {
    super();
    // Default chunk size: 4096 samples
    this.chunkSize = options?.processorOptions?.chunkSize || 4096;
    this.buffer = new Float32Array(0);
  }

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  process(inputs, outputs, parameters) {
    const input = inputs[0];

    // No input, skip processing
    if (!input || input.length === 0) {
      return true;
    }

    // Get the first channel (mono)
    const channelData = input[0];
    if (!channelData || channelData.length === 0) {
      return true;
    }

    // Append new data to buffer
    const newBuffer = new Float32Array(this.buffer.length + channelData.length);
    newBuffer.set(this.buffer);
    newBuffer.set(channelData, this.buffer.length);
    this.buffer = newBuffer;

    // Send chunks when buffer is large enough
    while (this.buffer.length >= this.chunkSize) {
      const chunk = this.buffer.slice(0, this.chunkSize);
      this.buffer = this.buffer.slice(this.chunkSize);

      // Send audio chunk to main thread
      this.port.postMessage({
        type: "audio",
        audio: chunk,
      });
    }

    return true;
  }
}

registerProcessor("audio-processor", AudioProcessor);
