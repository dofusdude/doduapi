package server

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/dofusdude/api/utils"
	"io"
	"io/fs"
	"log"
	"os"
	"strings"
	"sync"
)

type ImageContainerResult struct {
	ContainerId string
	Resolution  string
	AnkamaId    string
}

func RenderVectorImagesWorker(swfFiles []fs.DirEntry, ctx context.Context, resolution string, cli *client.Client, imgOutSubdirName string, done chan bool, result chan []string, yield chan string) {
	var containerIds []string
	path, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	for _, swfFile := range swfFiles {
		select {
		case <-done:
			return
		default:
			if swfFile.IsDir() || !strings.HasSuffix(swfFile.Name(), ".swf") {
				continue
			}

			absSwfPath := fmt.Sprintf("%s/data/vector/%s/%s", path, imgOutSubdirName, swfFile.Name())
			rawFileName := strings.TrimSuffix(swfFile.Name(), ".swf")
			finalImagePath := fmt.Sprintf("%s/data/img/%s/%s-%s.png", path, imgOutSubdirName, rawFileName, resolution)

			if _, err := os.Stat(finalImagePath); err == nil {
				yield <- finalImagePath
				continue
			}

			cmd := []string{"--screenshot", "last", "--screenshot-file", fmt.Sprintf("%s-%s.png", rawFileName, resolution), "-1", "-r1", "--width", resolution, "--height", resolution, swfFile.Name()}
			resp, err := cli.ContainerCreate(ctx, &container.Config{
				Image:      "stelzo/swf-renderer",
				Cmd:        cmd,
				Entrypoint: []string{"/usr/local/bin/dump-gnash"},
				Volumes: map[string]struct{}{
					"/home/developer": {},
				},
			}, &container.HostConfig{
				Binds:      []string{fmt.Sprintf("%s:/home/developer", fmt.Sprintf("%s/data/vector/%s", utils.DockerMountDataPath, imgOutSubdirName))},
				AutoRemove: true,
			}, nil, nil, "")
			if err != nil {
				panic(err)
			}

			if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
				panic(err)
			}

			statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
			select {
			case err := <-errCh:
				if err != nil && !strings.Contains(err.Error(), "Error response from daemon: No such container:") {
					panic(err)
				}
			case <-statusCh:
			}

			srcImagePath := fmt.Sprintf("%s/data/vector/%s/%s-%s.png", path, imgOutSubdirName, rawFileName, resolution)
			err = os.Rename(srcImagePath, finalImagePath)
			if err != nil {
				log.Println(err)
			}

			yield <- finalImagePath
			containerIds = append(containerIds, absSwfPath)
		}
	}

	result <- containerIds
}

func RenderVectorImages(done chan bool, imgOutSubdirName string) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}

	outPath := fmt.Sprintf("%s/data/vector/%s", path, imgOutSubdirName)
	ctx := context.Background()
	reader, err := cli.ImagePull(ctx, "stelzo/swf-renderer", types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}
	defer reader.Close()

	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		panic(err)
	}

	swfFiles, err := os.ReadDir(outPath)
	if err != nil {
		panic(err)
	}

	stopChan1 := make(chan bool)
	stopChan2 := make(chan bool)
	stopChan3 := make(chan bool)

	yieldChan1 := make(chan string)
	yieldChan2 := make(chan string)
	yieldChan3 := make(chan string)

	resChan1 := make(chan []string)
	resChan2 := make(chan []string)
	resChan3 := make(chan []string)

	go RenderVectorImagesWorker(swfFiles, ctx, utils.ImgResolutions[0], cli, imgOutSubdirName, stopChan1, resChan1, yieldChan1)
	go RenderVectorImagesWorker(swfFiles, ctx, utils.ImgResolutions[1], cli, imgOutSubdirName, stopChan2, resChan2, yieldChan2)
	go RenderVectorImagesWorker(swfFiles, ctx, utils.ImgResolutions[2], cli, imgOutSubdirName, stopChan3, resChan3, yieldChan3)

	worker1Done := false
	worker2Done := false
	worker3Done := false

	var origPaths [][]string

	var imageMutex sync.Mutex
	var pathsMutex sync.Mutex

	if utils.ImgWithResExists == nil {
		utils.ImgWithResExists = utils.NewSet()
	}

	for !worker1Done || !worker2Done || !worker3Done {
		select {
		case <-done:
			if !worker1Done {
				stopChan1 <- true
			}
			if !worker2Done {
				stopChan2 <- true
			}
			if !worker3Done {
				stopChan3 <- true
			}
			return
		case imgPath := <-yieldChan1:
			imageMutex.Lock()
			utils.ImgWithResExists.Add(imgPath)
			imageMutex.Unlock()
		case imgPath := <-yieldChan2:
			imageMutex.Lock()
			utils.ImgWithResExists.Add(imgPath)
			imageMutex.Unlock()
		case imgPath := <-yieldChan3:
			imageMutex.Lock()
			utils.ImgWithResExists.Add(imgPath)
			imageMutex.Unlock()
		case res1 := <-resChan1:
			pathsMutex.Lock()
			origPaths = append(origPaths, res1)
			pathsMutex.Unlock()
			worker1Done = true
		case res2 := <-resChan2:
			pathsMutex.Lock()
			origPaths = append(origPaths, res2)
			pathsMutex.Unlock()
			worker2Done = true
		case res3 := <-resChan3:
			pathsMutex.Lock()
			origPaths = append(origPaths, res3)
			pathsMutex.Unlock()
			worker3Done = true
		}
	}

	for _, subPaths := range origPaths {
		for _, path := range subPaths {
			_ = os.Remove(path)
		}
	}

	select {
	case <-done:
		return
	default:
		done <- true
	}
}
