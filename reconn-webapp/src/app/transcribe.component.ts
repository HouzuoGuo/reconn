import { HttpErrorResponse } from '@angular/common/http';
import { Component } from '@angular/core';
import { Observable, ReplaySubject, combineLatest, fromEvent, of, shareReplay } from 'rxjs';
import { catchError, exhaustMap, map, startWith, switchMap, take, tap } from 'rxjs/operators';
import { AudioRecorderService, ErrorCase, OutputFormat } from './audio_recorder.module';
import { ChatService, ReadbackResponse, ReadbackService, TranscribeRealTimeResponse } from './chat.module';

@Component({
  selector: 'transcribe',
  templateUrl: './transcribe.component.html',
})
export class TranscribeComponent {
  recordingInProgress = false;
  recordButtonCaption = 'Start recording';
  recording?: Blob;
  recorderError: Observable<ErrorCase>;
  transcribeStatus = new ReplaySubject<string>(1);

  constructor(readonly recorderService: AudioRecorderService, readonly chatService: ChatService) {
    this.recorderError = recorderService.recorderError.pipe(shareReplay({ bufferSize: 1, refCount: true }));
  }

  recordButtonClick() {
    if (this.recordingInProgress) {
      this.recorderService.stopRecording(OutputFormat.BLOB).then((blob) => {
        console.log('recording blob', blob);
        this.recording = blob as Blob;
        this.transcribeStatus.next("Recording size: " + this.recording.size / 1024 + "KB");
      });
      this.recordingInProgress = false;
      this.recordButtonCaption = 'Start over';
    } else {
      this.recorderService.startRecording();
      this.recordingInProgress = true;
      this.recordButtonCaption = 'Finish recording';
    }
  }

  transcribeButtonClick() {
    if (!this.recording) {
      return;
    }
    console.log('input recording', this.recording);
    this.transcribeStatus.next("Stand by, transcription is in progress.");
    this.chatService.transcribeRealTime(this.recording).pipe(
      map((resp: TranscribeRealTimeResponse) => {
        console.log('transcription response', resp);
        return 'Transcription: ' + resp.content;
      }),
      catchError((err: HttpErrorResponse) => {
        console.log('http error', err);
        return of('Error: ' + err.message);
      })
    ).subscribe((result: string) => {
      this.transcribeStatus.next(result);
    });
  }
}
