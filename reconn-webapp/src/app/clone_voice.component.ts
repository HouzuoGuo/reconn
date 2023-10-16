import { HttpErrorResponse } from '@angular/common/http';
import { Component } from '@angular/core';
import { Observable, ReplaySubject, combineLatest, fromEvent, of, shareReplay } from 'rxjs';
import { catchError, exhaustMap, map, startWith, switchMap, take, tap } from 'rxjs/operators';
import { AudioRecorderService, ErrorCase, OutputFormat } from './audio_recorder.module';
import { CloneRealtimeResponse, ReadbackResponse, ReadbackService, VoiceModelResponse, ChatService} from './chat.module';

@Component({
  selector: 'clone-voice',
  templateUrl: './clone_voice.component.html',
})
export class CloneVoiceComponent {
  userID = '';
  recordingInProgress = false;
  recordButtonCaption = 'Start recording';
  recording?: Blob;
  recorderError: Observable<ErrorCase>;
  cloneMessage = new ReplaySubject<string>(1);

  constructor(readonly recorderService: AudioRecorderService, private chatService: ChatService) {
    this.recorderError = recorderService.recorderError.pipe(shareReplay({ bufferSize: 1, refCount: true }));
  }

  recordButtonClick() {
    if (this.recordingInProgress) {
      this.recorderService.stopRecording(OutputFormat.BLOB).then((blob) => {
        console.log('recording blob', blob);
        this.recording = blob as Blob;
        this.cloneMessage.next("Recording size: " + this.recording.size / 1024 + "KB");
      });
      this.recordingInProgress = false;
      this.recordButtonCaption = 'Start over';
    } else {
      this.recorderService.startRecording();
      this.recordingInProgress = true;
      this.recordButtonCaption = 'Finish recording';
    }
  }

  cloneButtonClick() {
    if (!this.recording) {
      return;
    }
    console.log('input recording', this.recording);
    this.cloneMessage.next("Stand by, cloning is in progress.");
    this.chatService.cloneRealTime(this.userID, this.recording).pipe(
      map((cloneResp: CloneRealtimeResponse) => {
        console.log('clone response', cloneResp);
        return 'Cloned to model file: ' + cloneResp.model;
      }),
      catchError((err: HttpErrorResponse) => {
        console.log('http error', err);
        return of('Error: ' + err.message);
      })
    ).subscribe((result: string) => {
      this.cloneMessage.next(result);
    });
  }
}
