// Temporarily copied from https://github.com/killroywashere/ng-audio-recorder/tree/master, unknown license (copyright reserved to killroywashere?)
// TODO FIXME: switch to an open source alternative, or recreate the copyrighted code on my own.

import { EventEmitter, Injectable, NgModule } from '@angular/core';
import { connect } from 'extendable-media-recorder-wav-encoder';
import { MediaRecorder, register } from 'extendable-media-recorder';

export enum OutputFormat {
  URL,
  BLOB,
}

export enum ErrorCase {
  USER_CONSENT_FAILED,
  RECORDER_TIMEOUT,
  ALREADY_RECORDING
}

export enum RecorderState {
  INITIALIZING,
  INITIALIZED,
  RECORDING,
  PAUSED,
  STOPPING,
  STOPPED
}

@Injectable()
export class AudioRecorderService {
  private chunks: Array<any> = [];
  protected recorderEnded = new EventEmitter();
  public recorderError = new EventEmitter<ErrorCase>();
  private _recorderState = RecorderState.INITIALIZING;
  private firstTime = true;

  constructor() {
  }

  private recorder: any;


  private static guc() {
    return navigator.mediaDevices.getUserMedia({ audio: true });
  }


  getUserContent() {
    return AudioRecorderService.guc();
  }

  async startRecording() {
    if (this.firstTime) {
      await register(await connect());
      this.firstTime = false;
    }
    if (this._recorderState === RecorderState.RECORDING) {
      this.recorderError.emit(ErrorCase.ALREADY_RECORDING);
    }
    if (this._recorderState === RecorderState.PAUSED) {
      this.resume();
      return;
    }
    this._recorderState = RecorderState.INITIALIZING;
    AudioRecorderService.guc().then((mediaStream) => {
      this.recorder = new MediaRecorder(mediaStream, { mimeType: 'audio/wav' });
      this._recorderState = RecorderState.INITIALIZED;
      this.addListeners();
      this.recorder.start();
      this._recorderState = RecorderState.RECORDING;
    });
  }

  pause() {
    if (this._recorderState === RecorderState.RECORDING) {
      this.recorder.pause();
      this._recorderState = RecorderState.PAUSED;
    }
  }

  resume() {
    if (this._recorderState === RecorderState.PAUSED) {
      this._recorderState = RecorderState.RECORDING;
      this.recorder.resume();
    }
  }

  stopRecording(outputFormat: OutputFormat) {
    this._recorderState = RecorderState.STOPPING;
    return new Promise((resolve, reject) => {
      this.recorderEnded.subscribe((blob) => {
        this._recorderState = RecorderState.STOPPED;
        if (outputFormat === OutputFormat.BLOB) {
          resolve(blob);
        }
        if (outputFormat === OutputFormat.URL) {
          const audioURL = URL.createObjectURL(blob);
          resolve(audioURL);
        }
      }, _ => {
        this.recorderError.emit(ErrorCase.RECORDER_TIMEOUT);
        reject(ErrorCase.RECORDER_TIMEOUT);
      });
      this.recorder.stop();
    }).catch(() => {
      this.recorderError.emit(ErrorCase.USER_CONSENT_FAILED);
    });
  }

  getRecorderState() {
    return this._recorderState;
  }

  private addListeners() {
    this.recorder.ondataavailable = this.appendToChunks;
    this.recorder.onstop = this.recordingStopped;
  }

  private appendToChunks = (event: any) => {
    this.chunks.push(event.data);
  };
  private recordingStopped = (event: any) => {
    const blob = new Blob(this.chunks, { type: 'audio/wav' });
    this.chunks = [];
    this.recorderEnded.emit(blob);
    this.clear();
  };

  private clear() {
    this.recorder = null;
    this.chunks = [];
  }
}

@NgModule({
  declarations: [],
  imports: [
  ],
  exports: [],
  providers: [
    AudioRecorderService
  ]
})
export class AudioRecorderModule {}
